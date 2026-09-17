package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestUserCRUDRechargeAndBalance(t *testing.T) {
	handler := newTestServer()

	created := adminDo(t, handler, http.MethodPost, "/admin/users", map[string]any{"nickname": "Alice"})
	user := created["data"].(map[string]any)
	if user["id"].(float64) != 1 || user["status"] != "active" {
		t.Fatalf("unexpected user: %+v", user)
	}
	balance := user["balance"].(map[string]any)
	if balance["available_balance"] != "0.000000" || balance["frozen_balance"] != "0.000000" {
		t.Fatalf("unexpected balance: %+v", balance)
	}
	if _, ok := user["password_plaintext"]; ok {
		t.Fatal("password_plaintext must not be returned")
	}

	adminDo(t, handler, http.MethodPut, "/admin/users/1", map[string]any{"nickname": "Alice2", "user_group": "vip"})
	adminDo(t, handler, http.MethodPut, "/admin/users/1/status", map[string]any{"status": "suspended"})

	listed := adminDo(t, handler, http.MethodGet, "/admin/users", nil)
	list := listed["data"].(map[string]any)
	if list["total"].(float64) != 1 {
		t.Fatalf("users total = %v, want 1", list["total"])
	}

	recharged := adminDo(t, handler, http.MethodPost, "/admin/users/1/recharge", map[string]any{"amount": "50.5"})
	if recharged["data"].(map[string]any)["balance_after"] != "50.500000" {
		t.Fatalf("balance_after = %+v", recharged["data"])
	}

	balanceResp := adminDo(t, handler, http.MethodGet, "/admin/users/1/balance", nil)
	if balanceResp["data"].(map[string]any)["available_balance"] != "50.500000" {
		t.Fatalf("available_balance = %+v", balanceResp["data"])
	}

	txs := adminDo(t, handler, http.MethodGet, "/admin/users/1/balance-transactions", nil)
	txData := txs["data"].(map[string]any)
	if txData["total"].(float64) != 1 {
		t.Fatalf("transactions total = %v, want 1", txData["total"])
	}
	tx := txData["list"].([]any)[0].(map[string]any)
	if tx["tx_type"] != "recharge" || tx["amount"] != "50.500000" || tx["balance_after"] != "50.500000" {
		t.Fatalf("unexpected transaction: %+v", tx)
	}

	bad := adminRaw(t, handler, http.MethodPost, "/admin/users/1/recharge", map[string]any{"amount": "abc"})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid amount status = %d, want 400", bad.Code)
	}

	adminDo(t, handler, http.MethodDelete, "/admin/users/1", nil)
	missing := adminRaw(t, handler, http.MethodDelete, "/admin/users/1", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", missing.Code)
	}
}

func TestKeyLifecycleHidesPlaintext(t *testing.T) {
	handler := newTestServer()
	adminDo(t, handler, http.MethodPost, "/admin/users", map[string]any{"nickname": "Alice"})

	created := adminDo(t, handler, http.MethodPost, "/admin/users/1/keys", map[string]any{"key_name": "default", "prefix": "sk-"})
	keyData := created["data"].(map[string]any)
	fullKey, _ := keyData["full_key"].(string)
	if fullKey == "" {
		t.Fatalf("full_key missing: %+v", keyData)
	}
	keyID := int(keyData["id"].(float64))

	listed := adminDo(t, handler, http.MethodGet, "/admin/users/1/keys", nil)
	assertNoSecret(t, listed, fullKey)
	row := listed["data"].(map[string]any)["list"].([]any)[0].(map[string]any)
	if row["key_name"] != "default" || row["prefix"] != "sk-" || row["is_active"] != true {
		t.Fatalf("unexpected key row: %+v", row)
	}

	global := adminDo(t, handler, http.MethodGet, "/admin/keys", nil)
	assertNoSecret(t, global, fullKey)
	if global["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("global keys total = %+v", global["data"])
	}

	updated := adminDo(t, handler, http.MethodPut, "/admin/users/1/keys/"+itoa(keyID), map[string]any{"is_active": false})
	if updated["data"].(map[string]any)["is_active"] != false {
		t.Fatalf("key not deactivated: %+v", updated["data"])
	}

	reset := adminDo(t, handler, http.MethodPost, "/admin/users/1/keys/"+itoa(keyID)+"/reset", nil)
	newKey, _ := reset["data"].(map[string]any)["full_key"].(string)
	if newKey == "" || newKey == fullKey {
		t.Fatalf("reset did not produce a new key")
	}
	assertNoSecret(t, reset, fullKey)

	adminDo(t, handler, http.MethodDelete, "/admin/users/1/keys/"+itoa(keyID), nil)
	missing := adminRaw(t, handler, http.MethodDelete, "/admin/users/1/keys/"+itoa(keyID), nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("delete missing key status = %d, want 404", missing.Code)
	}
}

func TestListKeysMissingUserReturnsNotFound(t *testing.T) {
	res := adminRaw(t, newTestServer(), http.MethodGet, "/admin/users/404/keys", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestUserListReturnsFrozenBalanceFields(t *testing.T) {
	handler := newTestServer()
	adminDo(t, handler, http.MethodPost, "/admin/users", map[string]any{"nickname": "Alice"})

	listed := adminDo(t, handler, http.MethodGet, "/admin/users?page=1&page_size=20", nil)
	row := listed["data"].(map[string]any)["list"].([]any)[0].(map[string]any)
	balance := row["balance"].(map[string]any)
	if _, ok := balance["available_balance"]; !ok {
		t.Fatalf("missing available_balance: %+v", balance)
	}
	if _, ok := balance["frozen_balance"]; !ok {
		t.Fatalf("missing frozen_balance: %+v", balance)
	}
	if strings.Contains(row["nickname"].(string), "password") {
		t.Fatal("unexpected field")
	}
}
