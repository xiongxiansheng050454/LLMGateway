package proxy

import (
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"context"
)

// selectChannel returns the channel to use for a public model. Candidates come
// ordered by priority desc, weight desc, channel id; channels with a non-nil
// balance of zero or less are excluded. Within the highest priority group the
// choice is weighted-random using the injected source.
func (a *Service) selectChannel(model string) (domain.RouteCandidate, error) {
	candidates, err := a.orderedCandidates(model)
	if err != nil {
		return domain.RouteCandidate{}, err
	}
	if len(candidates) == 0 {
		return domain.RouteCandidate{}, ErrNoHealthyChannel
	}
	return candidates[0], nil
}

func (a *Service) orderedCandidates(model string) ([]domain.RouteCandidate, error) {
	result, err := a.store.RouteCandidates(model)
	if err != nil {
		return nil, err
	}
	candidates := []domain.RouteCandidate{}
	seen := map[int]bool{}
	for _, candidate := range result.List {
		if seen[candidate.ChannelID] {
			continue
		}
		health, healthErr := a.store.GetChannelHealth(candidate.ChannelID)
		if healthErr == nil && health.State == domain.HealthHalfOpen {
			allowed, probeErr := a.store.AcquireChannelProbe(context.Background(), candidate.ChannelID, a.requestTimeout)
			if probeErr != nil || !allowed {
				continue
			}
		}
		if candidate.Balance != nil {
			parsed, err := money.Parse6(*candidate.Balance)
			if err == nil && parsed.Cmp(0) <= 0 {
				continue
			}
		}
		seen[candidate.ChannelID] = true
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	highest := candidates[0].Priority
	group := []domain.RouteCandidate{}
	for _, candidate := range candidates {
		if candidate.Priority == highest {
			group = append(group, candidate)
		}
	}

	total := 0
	for _, candidate := range group {
		if candidate.Weight > 0 {
			total += candidate.Weight
		}
	}
	if total <= 0 {
		return append(group, candidates[len(group):]...), nil
	}

	pick := a.randIntN(total)
	for _, candidate := range group {
		if candidate.Weight <= 0 {
			continue
		}
		pick -= candidate.Weight
		if pick < 0 {
			ordered := []domain.RouteCandidate{candidate}
			for _, rest := range group {
				if rest.ChannelID != candidate.ChannelID {
					ordered = append(ordered, rest)
				}
			}
			for _, rest := range candidates {
				if rest.Priority != highest {
					ordered = append(ordered, rest)
				}
			}
			return ordered, nil
		}
	}
	return candidates, nil
}
