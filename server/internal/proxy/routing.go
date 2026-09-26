package proxy

import (
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/money"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// selectChannel returns the channel to use for a public model. Candidates come
// ordered by priority desc, weight desc, channel id; channels with a non-nil
// balance below the configured reserve are excluded. Within the highest priority
// group the choice is weighted-random using the injected source.
func (a *Service) selectChannel(ctx context.Context, model string) (catalog.RouteCandidate, error) {
	candidates, err := a.orderedCandidates(ctx, model)
	if err != nil {
		return catalog.RouteCandidate{}, err
	}
	if len(candidates) == 0 {
		return catalog.RouteCandidate{}, ErrNoHealthyChannel
	}
	return candidates[0], nil
}

func (a *Service) orderedCandidates(ctx context.Context, model string, stickyKey ...int) ([]catalog.RouteCandidate, error) {
	result, err := a.catalog.RouteCandidates(ctx, model)
	if err != nil {
		return nil, err
	}
	candidates := []catalog.RouteCandidate{}
	seen := map[int]bool{}
	for _, candidate := range result.List {
		if seen[candidate.ChannelID] {
			continue
		}
		health, healthErr := a.catalog.GetChannelHealth(ctx, candidate.ChannelID)
		if healthErr == nil && health.State == catalog.HealthHalfOpen {
			allowed, probeErr := a.catalog.AcquireChannelProbe(ctx, candidate.ChannelID, a.requestTimeout)
			if probeErr != nil || !allowed {
				continue
			}
		}
		if candidate.Balance != nil {
			parsed, err := money.Parse6(*candidate.Balance)
			if err == nil && (parsed.Cmp(0) <= 0 || parsed.Cmp(a.minRouteBalance) < 0) {
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
	group := []catalog.RouteCandidate{}
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

	pick := a.routePick(model, total, stickyKey...)
	for _, candidate := range group {
		if candidate.Weight <= 0 {
			continue
		}
		pick -= candidate.Weight
		if pick < 0 {
			ordered := []catalog.RouteCandidate{candidate}
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

// routePick keeps an API key on the same weighted candidate for a public model.
// A missing key falls back to the injected random source used by tests and
// unauthenticated internal callers.
func (a *Service) routePick(model string, total int, stickyKey ...int) int {
	if len(stickyKey) == 0 || stickyKey[0] <= 0 {
		return a.randIntN(total)
	}
	seed := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s", stickyKey[0], model)))
	return int(binary.BigEndian.Uint64(seed[:8]) % uint64(total))
}
