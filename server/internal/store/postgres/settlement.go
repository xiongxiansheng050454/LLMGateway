package postgres

import (
	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/usage"
)

func (t *Tx) LockChannel(channelID int) error {
	_, err := t.queries.LockChannel(t.ctx, int64(channelID))
	return mapError(err)
}

func (t *Tx) GetChannelBalanceText(channelID int) (string, error) {
	row, err := t.queries.GetChannel(t.ctx, int64(channelID))
	if err != nil {
		return "", mapError(err)
	}
	return textValue(row.Balance), nil
}

func (t *Tx) UpdateChannelBalance(channelID int, balance string) (bool, error) {
	affected, err := t.queries.UpdateChannelBalance(t.ctx, sqlc.UpdateChannelBalanceParams{Balance: balance, ID: int64(channelID)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) SettleQuotaReservation(reservationID int64, requestID string, userID, keyID int, actualTokens int64, actualCost string) error {
	return settleQuotaTx(t.ctx, t.tx, reservationID, requestID, userID, keyID, actualTokens, actualCost, t.now())
}

func (t *Tx) InsertUsageLog(in usage.UsageLogInput) (int, error) {
	id, err := insertUsageLog(t.ctx, t.queries, in)
	if err != nil {
		return 0, mapError(err)
	}
	return id, nil
}
