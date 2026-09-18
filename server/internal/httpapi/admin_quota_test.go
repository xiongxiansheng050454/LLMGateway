package httpapi

import (
	"net/http"
	"testing"
)

func TestAdminQuotaPolicyCRUD(t *testing.T) {
	handler := newTestServer()
	create := adminDo(t, handler, http.MethodPost, "/admin/quota-policies", map[string]any{"policy_name": "daily", "scope_type": "user", "scope_id": 1, "period_type": "day", "token_limit": 100, "cost_limit": "2.5"})
	if create["data"].(map[string]any)["cost_limit"] != "2.500000" {
		t.Fatalf("create response = %+v", create)
	}
	list := adminDo(t, handler, http.MethodGet, "/admin/quota-policies?scope_type=user&scope_id=1", nil)
	if list["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("list response = %+v", list)
	}
	update := adminDo(t, handler, http.MethodPut, "/admin/quota-policies/1", map[string]any{"enabled": false})
	if update["data"].(map[string]any)["enabled"] != false {
		t.Fatalf("update response = %+v", update)
	}
	adminDo(t, handler, http.MethodDelete, "/admin/quota-policies/1", nil)
}
