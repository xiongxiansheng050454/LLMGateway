package proxy

import (
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
)

// selectChannel returns the channel to use for a public model. Candidates come
// ordered by priority desc, weight desc, channel id; channels with a non-nil
// balance of zero or less are excluded. Within the highest priority group the
// choice is weighted-random using the injected source.
func (a *Service) selectChannel(model string) (domain.RouteCandidate, error) {
	result, err := a.store.RouteCandidates(model)
	if err != nil {
		return domain.RouteCandidate{}, err
	}

	candidates := []domain.RouteCandidate{}
	for _, candidate := range result.List {
		if candidate.Balance != nil {
			parsed, err := money.Parse6(*candidate.Balance)
			if err == nil && parsed.Cmp(0) <= 0 {
				continue
			}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		// No mapping, no enabled channel, or every candidate is tripped open.
		return domain.RouteCandidate{}, ErrNoHealthyChannel
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
		return group[0], nil
	}

	pick := a.randIntN(total)
	for _, candidate := range group {
		if candidate.Weight <= 0 {
			continue
		}
		pick -= candidate.Weight
		if pick < 0 {
			return candidate, nil
		}
	}
	return group[len(group)-1], nil
}
