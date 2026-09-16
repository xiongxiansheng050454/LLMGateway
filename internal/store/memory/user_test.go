package memory

import (
	"errors"
	"testing"

	"LLMGateway/internal/crypto"
	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
)

func TestUserCRUDAndBalance(t *testing.T) {
	st := New()

	created, err := st.CreateUser(domain.UserInput{Nickname: "Alice"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created["id"] != 1 || created["user_group"] != "default" || created["status"] != "active" {
		t.Fatalf("unexpected user: %+v", created)
	}
	balance := created["balance"].(map[string]any)
	if balance["available_balance"] != "0.000000" || balance["frozen_balance"] != "0.000000" {
		t.Fatalf("unexpected initial balance: %+v", balance)
	}

	if _, err := st.UpdateUser(1, domain.UserInput{Nickname: "Alice2", UserGroup: "vip"}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if _, err := st.UpdateUserStatus(1, "suspended"); err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if _, err := st.UpdateUserStatus(1, "bogus"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid status err = %v, want ErrInvalid", err)
	}

	listed, err := st.ListUsers(1, 20)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if listed.Total != 1 {
		t.Fatalf("ListUsers total = %d, want 1", listed.Total)
	}
	row := listed.List[0].(map[string]any)
	if row["nickname"] != "Alice2" || row["user_group"] != "vip" || row["status"] != "suspended" {
		t.Fatalf("unexpected listed user: %+v", row)
	}

	if _, err := st.UpdateUser(404, domain.UserInput{Nickname: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("UpdateUser missing err = %v, want ErrNotFound", err)
	}
	if err := st.DeleteUser(1); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if err := st.DeleteUser(1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteUser err = %v, want ErrNotFound", err)
	}
}

func TestRechargeNormalizesAndIsIdempotent(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}

	result, err := st.RechargeUser(1, domain.RechargeInput{Amount: "50.5"})
	if err != nil {
		t.Fatalf("RechargeUser: %v", err)
	}
	if result["balance_after"] != "50.500000" {
		t.Fatalf("balance_after = %v, want 50.500000", result["balance_after"])
	}

	balance, err := st.GetUserBalance(1)
	if err != nil {
		t.Fatalf("GetUserBalance: %v", err)
	}
	if balance["available_balance"] != "50.500000" {
		t.Fatalf("available_balance = %v", balance["available_balance"])
	}

	// Same related_order_id must not double-credit.
	first, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000", RelatedOrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first["balance_after"] != second["balance_after"] {
		t.Fatalf("idempotent recharge mismatch: %v vs %v", first["balance_after"], second["balance_after"])
	}
	balance, _ = st.GetUserBalance(1)
	if balance["available_balance"] != "60.500000" {
		t.Fatalf("available_balance = %v, want 60.500000", balance["available_balance"])
	}

	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "abc"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid amount err = %v, want ErrInvalid", err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "-1.000000"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("negative amount err = %v, want ErrInvalid", err)
	}
	if _, err := st.RechargeUser(404, domain.RechargeInput{Amount: "1.000000"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}

	txs, err := st.ListBalanceTransactions(1, 1, 20)
	if err != nil {
		t.Fatalf("ListBalanceTransactions: %v", err)
	}
	if txs.Total != 2 {
		t.Fatalf("transactions total = %d, want 2", txs.Total)
	}
}

func TestKeysLifecycleHidesPlaintext(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}

	created, err := st.CreateKey(1, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	fullKey, _ := created["full_key"].(string)
	if fullKey == "" {
		t.Fatalf("full_key missing: %+v", created)
	}
	keyID := created["id"].(int)

	// The stored record must keep only the hash.
	key := st.keys[keyID]
	if key.keyHash != crypto.HashKey(fullKey) || key.keyHash == fullKey {
		t.Fatalf("key hash mismatch or plaintext stored")
	}

	listed, err := st.ListUserKeys(1, 1, 20)
	if err != nil {
		t.Fatalf("ListUserKeys: %v", err)
	}
	row := listed.List[0].(map[string]any)
	if _, ok := row["full_key"]; ok {
		t.Fatalf("list leaked full_key: %+v", row)
	}
	if row["key_name"] != "default" || row["prefix"] != "sk-" || row["is_active"] != true {
		t.Fatalf("unexpected key row: %+v", row)
	}

	global, err := st.ListKeys(1, 20)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if global.Total != 1 {
		t.Fatalf("global keys total = %d, want 1", global.Total)
	}

	if _, err := st.UpdateKey(1, keyID, domain.KeyUpdateInput{IsActive: boolPtr(false)}); err != nil {
		t.Fatalf("UpdateKey: %v", err)
	}
	listed, _ = st.ListUserKeys(1, 1, 20)
	if listed.List[0].(map[string]any)["is_active"] != false {
		t.Fatal("key was not deactivated")
	}

	reset, err := st.ResetKey(1, keyID)
	if err != nil {
		t.Fatalf("ResetKey: %v", err)
	}
	newKey, _ := reset["full_key"].(string)
	if newKey == "" || newKey == fullKey {
		t.Fatalf("reset did not produce a new key: %q", newKey)
	}
	if st.keys[keyID].keyHash != crypto.HashKey(newKey) {
		t.Fatal("reset did not rotate the stored hash")
	}

	if err := st.DeleteKey(1, keyID); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if err := st.DeleteKey(1, keyID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteKey err = %v, want ErrNotFound", err)
	}
	if _, err := st.CreateKey(404, domain.KeyInput{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("CreateKey missing user err = %v, want ErrNotFound", err)
	}
}

func TestDeleteUserRemovesKeysAndTransactions(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "5.000000"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateKey(1, domain.KeyInput{}); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteUser(1); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	keys, _ := st.ListKeys(1, 20)
	if keys.Total != 0 {
		t.Fatalf("keys not cascade deleted: %d", keys.Total)
	}
	if _, err := st.GetUserBalance(1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("balance not deleted: %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
