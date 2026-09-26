package storefake

import (
	"context"
	"errors"
	"testing"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/store"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func newAccounts(st *Store) *accounts.Server { return accounts.New(st, st.AccountsTx()) }

func TestUserCRUDAndBalance(t *testing.T) {
	st := New()
	acc := newAccounts(st)

	created, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "Alice"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.ID != 1 || created.UserGroup != "default" || created.Status != "active" {
		t.Fatalf("unexpected user: %+v", created)
	}
	balance := created.Balance
	if balance.AvailableBalance != "0.000000" || balance.FrozenBalance != "0.000000" {
		t.Fatalf("unexpected initial balance: %+v", balance)
	}

	if _, err := acc.UpdateUser(context.Background(), 1, domain.UserInput{Nickname: "Alice2", UserGroup: "vip"}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if _, err := acc.UpdateUserStatus(context.Background(), 1, "suspended"); err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if _, err := acc.UpdateUserStatus(context.Background(), 1, "bogus"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid status err = %v, want ErrInvalid", err)
	}

	listed, err := st.ListUsers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if listed.Total != 1 {
		t.Fatalf("ListUsers total = %d, want 1", listed.Total)
	}
	row := listed.List[0]
	if row.Nickname != "Alice2" || row.UserGroup != "vip" || row.Status != "suspended" {
		t.Fatalf("unexpected listed user: %+v", row)
	}

	if _, err := acc.UpdateUser(context.Background(), 404, domain.UserInput{Nickname: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("UpdateUser missing err = %v, want ErrNotFound", err)
	}
	if err := acc.DeleteUser(context.Background(), 1); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if err := acc.DeleteUser(context.Background(), 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteUser err = %v, want ErrNotFound", err)
	}
}

func TestRechargeNormalizesAndIsIdempotent(t *testing.T) {
	st := New()
	acc := newAccounts(st)
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}

	result, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "50.5"})
	if err != nil {
		t.Fatalf("RechargeUser: %v", err)
	}
	if result.BalanceAfter != "50.500000" {
		t.Fatalf("balance_after = %v, want 50.500000", result.BalanceAfter)
	}

	balance, err := st.GetUserBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUserBalance: %v", err)
	}
	if balance.AvailableBalance != "50.500000" {
		t.Fatalf("available_balance = %v", balance.AvailableBalance)
	}

	// Same related_order_id must not double-credit.
	first, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.BalanceAfter != second.BalanceAfter {
		t.Fatalf("idempotent recharge mismatch: %v vs %v", first.BalanceAfter, second.BalanceAfter)
	}
	balance, _ = st.GetUserBalance(context.Background(), 1)
	if balance.AvailableBalance != "60.500000" {
		t.Fatalf("available_balance = %v, want 60.500000", balance.AvailableBalance)
	}

	if _, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "abc"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid amount err = %v, want ErrInvalid", err)
	}
	if _, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "-1.000000"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("negative amount err = %v, want ErrInvalid", err)
	}
	if _, err := acc.RechargeUser(context.Background(), 404, domain.RechargeInput{Amount: "1.000000"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}

	txs, err := st.ListBalanceTransactions(context.Background(), 1, 1, 20)
	if err != nil {
		t.Fatalf("ListBalanceTransactions: %v", err)
	}
	if txs.Total != 2 {
		t.Fatalf("transactions total = %d, want 2", txs.Total)
	}
}

func TestKeysLifecycleHidesPlaintext(t *testing.T) {
	st := New()
	acc := newAccounts(st)
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}

	created, err := acc.CreateKey(context.Background(), 1, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	fullKey := created.FullKey
	if fullKey == "" {
		t.Fatalf("full_key missing: %+v", created)
	}
	keyID := created.ID

	// The stored record must keep only the hash.
	key := st.keys[keyID]
	if key.keyHash != crypto.HashKey(fullKey) || key.keyHash == fullKey {
		t.Fatalf("key hash mismatch or plaintext stored")
	}

	listed, err := st.ListUserKeys(context.Background(), 1, 1, 20)
	if err != nil {
		t.Fatalf("ListUserKeys: %v", err)
	}
	row := listed.List[0]
	if row.KeyName != "default" || row.Prefix != "sk-" || row.IsActive != true {
		t.Fatalf("unexpected key row: %+v", row)
	}

	global, err := st.ListKeys(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if global.Total != 1 {
		t.Fatalf("global keys total = %d, want 1", global.Total)
	}

	if _, err := acc.UpdateKey(context.Background(), 1, keyID, domain.KeyUpdateInput{IsActive: boolPtr(false)}); err != nil {
		t.Fatalf("UpdateKey: %v", err)
	}
	listed, _ = st.ListUserKeys(context.Background(), 1, 1, 20)
	if listed.List[0].IsActive != false {
		t.Fatal("key was not deactivated")
	}

	reset, err := acc.ResetKey(context.Background(), 1, keyID)
	if err != nil {
		t.Fatalf("ResetKey: %v", err)
	}
	newKey := reset.FullKey
	if newKey == "" || newKey == fullKey {
		t.Fatalf("reset did not produce a new key: %q", newKey)
	}
	if st.keys[keyID].keyHash != crypto.HashKey(newKey) {
		t.Fatal("reset did not rotate the stored hash")
	}

	if err := acc.DeleteKey(context.Background(), 1, keyID); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if err := acc.DeleteKey(context.Background(), 1, keyID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteKey err = %v, want ErrNotFound", err)
	}
	if _, err := acc.CreateKey(context.Background(), 404, domain.KeyInput{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("CreateKey missing user err = %v, want ErrNotFound", err)
	}
}

func TestListUserKeysMissingUserReturnsNotFound(t *testing.T) {
	st := New()
	if _, err := st.ListUserKeys(context.Background(), 404, 1, 20); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ListUserKeys missing user err = %v, want ErrNotFound", err)
	}
}

func TestCreateKeyNormalizesExpiresAtToUTC(t *testing.T) {
	st := New()
	acc := newAccounts(st)
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	created, err := acc.CreateKey(context.Background(), 1, domain.KeyInput{ExpiresAt: "2027-01-01T00:00:00+08:00"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	keyID := created.ID

	listed, err := st.ListUserKeys(context.Background(), 1, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	row := listed.List[0]
	if row.ExpiresAt == nil || *row.ExpiresAt != "2026-12-31T16:00:00Z" {
		t.Fatalf("expires_at = %v, want 2026-12-31T16:00:00Z (UTC)", row.ExpiresAt)
	}
	if keyID == 0 {
		t.Fatal("unexpected key id")
	}
}

func TestRechargeOrderScopedPerUser(t *testing.T) {
	st := New()
	acc := newAccounts(st)
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "B"}); err != nil {
		t.Fatal(err)
	}

	if _, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "1.000000", RelatedOrderID: "order-x"}); err != nil {
		t.Fatalf("user 1 order: %v", err)
	}
	if _, err := acc.RechargeUser(context.Background(), 2, domain.RechargeInput{Amount: "1.000000", RelatedOrderID: "order-x"}); err != nil {
		t.Fatalf("same order id for another user should be allowed: %v", err)
	}
}

func TestDeleteUserRemovesKeysAndTransactions(t *testing.T) {
	st := New()
	acc := newAccounts(st)
	if _, err := acc.CreateUser(context.Background(), domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := acc.RechargeUser(context.Background(), 1, domain.RechargeInput{Amount: "5.000000"}); err != nil {
		t.Fatal(err)
	}
	if _, err := acc.CreateKey(context.Background(), 1, domain.KeyInput{}); err != nil {
		t.Fatal(err)
	}

	if err := acc.DeleteUser(context.Background(), 1); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	keys, _ := st.ListKeys(context.Background(), 1, 20)
	if keys.Total != 0 {
		t.Fatalf("keys not cascade deleted: %d", keys.Total)
	}
	if _, err := st.GetUserBalance(context.Background(), 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("balance not deleted: %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
