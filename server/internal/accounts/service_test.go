package accounts_test

import (
	"errors"
	"testing"

	"LLMGateway/server/internal/store"
	"LLMGateway/server/internal/testutil/app"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func boolPtr(value bool) *bool { return &value }

func TestServiceUserLifecycle(t *testing.T) {
	deps := app.New()
	acc := deps.Accounts

	created, err := acc.CreateUser(domain.UserInput{Nickname: "Alice"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.UserGroup != "default" || created.Status != "active" {
		t.Fatalf("defaults not applied: %+v", created)
	}
	if created.Balance.AvailableBalance != "0.000000" {
		t.Fatalf("initial balance = %+v", created.Balance)
	}
	if _, err := acc.CreateUser(domain.UserInput{Status: "bogus"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid status err = %v, want ErrInvalid", err)
	}

	updated, err := acc.UpdateUser(created.ID, domain.UserInput{Nickname: "Alice2", UserGroup: "vip"})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Nickname != "Alice2" || updated.UserGroup != "vip" {
		t.Fatalf("unexpected update: %+v", updated)
	}

	suspended, err := acc.UpdateUserStatus(created.ID, "suspended")
	if err != nil || suspended.Status != "suspended" {
		t.Fatalf("UpdateUserStatus = %+v, %v", suspended, err)
	}
	if _, err := acc.UpdateUserStatus(created.ID, "bogus"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid status err = %v, want ErrInvalid", err)
	}
	if _, err := acc.UpdateUser(404, domain.UserInput{Nickname: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}
}

func TestServiceRechargeAndDebit(t *testing.T) {
	deps := app.New()
	acc := deps.Accounts

	user, err := acc.CreateUser(domain.UserInput{Nickname: "Alice"})
	if err != nil {
		t.Fatal(err)
	}

	recharged, err := acc.RechargeUser(user.ID, domain.RechargeInput{Amount: "50.5"})
	if err != nil {
		t.Fatalf("RechargeUser: %v", err)
	}
	if recharged.BalanceAfter != "50.500000" {
		t.Fatalf("balance_after = %v, want 50.500000", recharged.BalanceAfter)
	}

	// Idempotent by related_order_id.
	first, err := acc.RechargeUser(user.ID, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := acc.RechargeUser(user.ID, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.BalanceAfter != second.BalanceAfter {
		t.Fatalf("idempotent recharge mismatch: %v vs %v", first.BalanceAfter, second.BalanceAfter)
	}

	if _, err := acc.RechargeUser(user.ID, domain.RechargeInput{Amount: "abc"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid amount err = %v, want ErrInvalid", err)
	}

	debit, err := acc.DebitUserBalance(user.ID, "3.5", "request")
	if err != nil {
		t.Fatalf("DebitUserBalance: %v", err)
	}
	if debit.BalanceAfter != "57.000000" {
		t.Fatalf("balance_after = %v, want 57.000000", debit.BalanceAfter)
	}
	if _, err := acc.DebitUserBalance(user.ID, "1000.000000", "too much"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("insufficient err = %v, want ErrInvalid", err)
	}
	if _, err := acc.DebitUserBalance(404, "1.000000", "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}
}

func TestServiceKeyLifecycle(t *testing.T) {
	deps := app.New()
	acc := deps.Accounts

	user, err := acc.CreateUser(domain.UserInput{Nickname: "Alice"})
	if err != nil {
		t.Fatal(err)
	}

	key, err := acc.CreateKey(user.ID, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if key.FullKey == "" {
		t.Fatal("full_key missing")
	}
	if _, err := acc.CreateKey(404, domain.KeyInput{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("CreateKey missing user err = %v, want ErrNotFound", err)
	}

	updated, err := acc.UpdateKey(user.ID, key.ID, domain.KeyUpdateInput{IsActive: boolPtr(false)})
	if err != nil {
		t.Fatalf("UpdateKey: %v", err)
	}
	if updated.IsActive {
		t.Fatal("key not deactivated")
	}
	if _, err := acc.UpdateKey(user.ID, key.ID, domain.KeyUpdateInput{}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("missing is_active err = %v, want ErrInvalid", err)
	}

	reset, err := acc.ResetKey(user.ID, key.ID)
	if err != nil {
		t.Fatalf("ResetKey: %v", err)
	}
	if reset.FullKey == "" || reset.FullKey == key.FullKey {
		t.Fatal("reset did not rotate the key")
	}

	if err := acc.DeleteKey(user.ID, key.ID); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if err := acc.DeleteKey(user.ID, key.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteKey err = %v, want ErrNotFound", err)
	}

	if err := acc.DeleteUser(user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if err := acc.DeleteUser(user.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteUser err = %v, want ErrNotFound", err)
	}
}
