package accounts

import (
	"context"
	"fmt"

	apperrors "LLMGateway/server/internal/errors"
	"LLMGateway/server/internal/money"
)

const (
	defaultUserGroup  = "default"
	defaultUserStatus = "active"
)

// CreateUser applies account defaults and validation, then inserts the user and
// its balance row in one transaction.
func (a *Server) CreateUser(in UserInput) (UserDTO, error) {
	if in.UserGroup == "" {
		in.UserGroup = defaultUserGroup
	}
	if in.Status == "" {
		in.Status = defaultUserStatus
	}
	if in.Status != "active" && in.Status != "suspended" {
		return UserDTO{}, fmt.Errorf("%w: invalid status", apperrors.ErrInvalid)
	}

	var created User
	err := a.tx.InTx(context.Background(), func(tx Tx) error {
		user, err := tx.InsertUser(in.Nickname, in.UserGroup, in.Status)
		if err != nil {
			return err
		}
		if err := tx.InsertUserBalance(user.ID); err != nil {
			return err
		}
		created = user
		return nil
	})
	if err != nil {
		return UserDTO{}, err
	}
	return userDTO(created), nil
}

// UpdateUser merges the requested fields with the current user and returns the
// updated record.
func (a *Server) UpdateUser(id int, in UserInput) (UserDTO, error) {
	var updated User
	err := a.tx.InTx(context.Background(), func(tx Tx) error {
		current, err := tx.GetUser(id)
		if err != nil {
			return err
		}
		group := in.UserGroup
		if group == "" {
			group = current.UserGroup
		}
		user, ok, err := tx.UpdateUser(id, in.Nickname, group)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrNotFound
		}
		updated = user
		return nil
	})
	if err != nil {
		return UserDTO{}, err
	}
	return userDTO(updated), nil
}

// UpdateUserStatus validates and applies a user status change.
func (a *Server) UpdateUserStatus(id int, status string) (UserDTO, error) {
	if status != "active" && status != "suspended" {
		return UserDTO{}, fmt.Errorf("%w: invalid status", apperrors.ErrInvalid)
	}

	var updated User
	err := a.tx.InTx(context.Background(), func(tx Tx) error {
		user, ok, err := tx.UpdateUserStatus(id, status)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrNotFound
		}
		updated = user
		return nil
	})
	if err != nil {
		return UserDTO{}, err
	}
	return userDTO(updated), nil
}

// DeleteUser releases the user's quota reservations and removes the user in one
// transaction.
func (a *Server) DeleteUser(id int) error {
	return a.tx.InTx(context.Background(), func(tx Tx) error {
		if err := tx.DeleteQuotaReservationsForUser(id); err != nil {
			return err
		}
		ok, err := tx.DeleteUser(id)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

// RechargeUser credits the available balance and records a ledger entry. The
// balance row lock precedes the idempotency lookup so concurrent repeats of the
// same related_order_id serialize and return the original result.
func (a *Server) RechargeUser(id int, in RechargeInput) (BalanceUpdateDTO, error) {
	amount, err := money.Parse6(in.Amount)
	if err != nil || amount.Cmp(0) <= 0 {
		return BalanceUpdateDTO{}, fmt.Errorf("%w: invalid amount", apperrors.ErrInvalid)
	}

	var balanceAfter string
	err = a.tx.InTx(context.Background(), func(tx Tx) error {
		if err := tx.LockUserBalance(id); err != nil {
			return err
		}
		if in.RelatedOrderID != "" {
			existing, found, err := tx.GetBalanceTransactionByOrder(id, in.RelatedOrderID)
			if err != nil {
				return err
			}
			if found {
				balanceAfter = existing
				return nil
			}
		}
		currentText, err := tx.GetUserBalanceText(id)
		if err != nil {
			return err
		}
		current, err := money.Parse6(currentText)
		if err != nil {
			return fmt.Errorf("%w: invalid balance", apperrors.ErrInvalid)
		}
		next := money.Format6(current.Add(amount))
		ok, err := tx.UpdateUserBalance(id, next)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrNotFound
		}
		if err := tx.InsertBalanceTransaction(BalanceTransactionInput{
			UserID:         id,
			TxType:         "recharge",
			Amount:         money.Format6(amount),
			BalanceAfter:   next,
			RelatedOrderID: in.RelatedOrderID,
			Description:    in.Description,
		}); err != nil {
			return err
		}
		balanceAfter = next
		return nil
	})
	if err != nil {
		return BalanceUpdateDTO{}, err
	}
	return BalanceUpdateDTO{BalanceAfter: balanceAfter}, nil
}

// DebitUserBalance deducts amount (6 decimals) from the available balance and
// records a consume transaction. Insufficient balance returns ErrInvalid; a
// missing user returns ErrNotFound.
func (a *Server) DebitUserBalance(userID int, amount string, description string) (BalanceUpdateDTO, error) {
	parsed, err := money.Parse6(amount)
	if err != nil || parsed.Cmp(0) <= 0 {
		return BalanceUpdateDTO{}, fmt.Errorf("%w: invalid amount", apperrors.ErrInvalid)
	}

	var balanceAfter string
	err = a.tx.InTx(context.Background(), func(tx Tx) error {
		if err := tx.LockUserBalance(userID); err != nil {
			return err
		}
		currentText, err := tx.GetUserBalanceText(userID)
		if err != nil {
			return err
		}
		current, err := money.Parse6(currentText)
		if err != nil {
			return fmt.Errorf("%w: invalid balance", apperrors.ErrInvalid)
		}
		if current.Cmp(parsed) < 0 {
			return fmt.Errorf("%w: insufficient balance", apperrors.ErrInvalid)
		}
		next := money.Format6(current.Sub(parsed))
		ok, err := tx.UpdateUserBalance(userID, next)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrNotFound
		}
		if err := tx.InsertBalanceTransaction(BalanceTransactionInput{
			UserID:       userID,
			TxType:       "consume",
			Amount:       money.Format6(parsed),
			BalanceAfter: next,
			Description:  description,
		}); err != nil {
			return err
		}
		balanceAfter = next
		return nil
	})
	if err != nil {
		return BalanceUpdateDTO{}, err
	}
	return BalanceUpdateDTO{BalanceAfter: balanceAfter}, nil
}

func userDTO(user User) UserDTO {
	return UserDTO{
		ID:        user.ID,
		Nickname:  user.Nickname,
		UserGroup: user.UserGroup,
		Status:    user.Status,
		Balance:   BalanceDTO{AvailableBalance: user.AvailableBalance, FrozenBalance: user.FrozenBalance},
	}
}
