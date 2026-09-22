package catalog

import (
	"context"
	"time"
)

func (a *Server) GetChannelHealth(channelID int) (ChannelHealth, error) {
	row, found, err := a.health.GetChannelHealthRow(channelID)
	if err != nil {
		return ChannelHealth{}, err
	}
	if !found {
		return NewChannelHealth(channelID), nil
	}
	return EvaluateChannelHealth(row, a.now(), a.breaker), nil
}

func (a *Server) RecordChannelSuccess(channelID int) (ChannelHealth, error) {
	return a.recordChannelHealth(channelID, func(current ChannelHealth) ChannelHealth {
		return ApplyChannelSuccess(current, a.now())
	})
}

func (a *Server) RecordChannelFailure(channelID int, reason FailureReason) (ChannelHealth, error) {
	if !reason.CountsAsChannelFailure() {
		return a.GetChannelHealth(channelID)
	}
	return a.recordChannelHealth(channelID, func(current ChannelHealth) ChannelHealth {
		return ApplyChannelFailure(current, reason, a.now(), a.breaker)
	})
}

func (a *Server) RecordChannelAttempt(ctx context.Context, channelID int, success bool, reason FailureReason) (ChannelHealth, error) {
	if success {
		return a.RecordChannelSuccess(channelID)
	}
	return a.RecordChannelFailure(channelID, reason)
}

func (a *Server) ResetChannelHealth(channelID int) error {
	return a.tx.InTx(context.Background(), func(tx Tx) error {
		return tx.DeleteChannelHealth(channelID)
	})
}

func (a *Server) ListChannelHealth() (ListResponse[ChannelHealthDTO], error) {
	rows, err := a.health.ListChannelHealthRows()
	if err != nil {
		return ListResponse[ChannelHealthDTO]{}, err
	}
	list := []ChannelHealthDTO{}
	for i := range rows {
		health := EvaluateChannelHealth(rows[i], a.now(), a.breaker)
		list = append(list, channelHealthDTO(&health))
	}
	return ListResponse[ChannelHealthDTO]{List: list, Total: len(list)}, nil
}

func (a *Server) AcquireChannelProbe(ctx context.Context, channelID int, lease time.Duration) (bool, error) {
	return a.health.AcquireChannelProbe(ctx, channelID, lease)
}

// recordChannelHealth applies a state transition inside a transaction, holding
// the row lock so concurrent recordings cannot lose updates.
func (a *Server) recordChannelHealth(channelID int, apply func(ChannelHealth) ChannelHealth) (ChannelHealth, error) {
	var next ChannelHealth
	err := a.tx.InTx(context.Background(), func(tx Tx) error {
		if err := tx.EnsureChannelHealth(channelID); err != nil {
			return err
		}
		row, err := tx.GetChannelHealthForUpdate(channelID)
		if err != nil {
			return err
		}
		next = apply(EvaluateChannelHealth(row, a.now(), a.breaker))
		ok, err := tx.UpdateChannelHealth(next)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return ChannelHealth{}, err
	}
	return next, nil
}

func channelHealthDTO(health *ChannelHealth) ChannelHealthDTO {
	return ChannelHealthDTO{
		ChannelID:           health.ChannelID,
		State:               string(health.State),
		ConsecutiveFailures: health.ConsecutiveFailures,
		SuccessCount:        health.SuccessCount,
		FailureCount:        health.FailureCount,
		OpenedAt:            health.OpenedAt,
		UpdatedAt:           health.UpdatedAt,
	}
}
