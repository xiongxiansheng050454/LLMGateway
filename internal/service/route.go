package service

import (
	"LLMGateway/internal/money"
)

// RouteCandidate is a channel that can serve a public model.
type RouteCandidate struct {
	ChannelID     int
	ChannelName   string
	UpstreamModel string
	Priority      int
	Weight        int
	Balance       *string
}

// SelectChannel returns the channel to use for a public model. Candidates come
// ordered by priority desc, weight desc, channel id; channels with a non-nil
// balance of zero or less are excluded. Within the highest priority group the
// choice is weighted-random using the injected source.
func (p *Proxy) SelectChannel(model string) (RouteCandidate, error) {
	result, err := p.store.RouteCandidates(model)
	if err != nil {
		return RouteCandidate{}, err
	}

	candidates := []RouteCandidate{}
	for _, item := range result.List {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		candidate := RouteCandidate{
			ChannelID:     toInt(entry["channel_id"]),
			ChannelName:   toString(entry["channel_name"]),
			UpstreamModel: toString(entry["upstream_model"]),
			Priority:      toInt(entry["priority"]),
			Weight:        toInt(entry["weight"]),
		}
		if balance, ok := entry["balance"].(*string); ok {
			candidate.Balance = balance
			if balance != nil {
				parsed, err := money.Parse6(*balance)
				if err == nil && parsed.Cmp(0) <= 0 {
					continue
				}
			}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return RouteCandidate{}, ErrNoChannel
	}

	highest := candidates[0].Priority
	group := []RouteCandidate{}
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

	pick := p.randIntN(total)
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
