package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestChannelCRUDDoesNotLeakAPIKey(t *testing.T) {
	handler := newTestHandler()

	created := adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{
		"name": "OpenAI", "base_url": "https://api.openai.test", "api_key": "sk-secret", "auth_type": "bearer", "status": 1, "weight": 100, "priority": 10, "balance": "100.000000",
	})
	assertNoSecret(t, created, "sk-secret")
	channel := created["data"].(map[string]any)
	if channel["id"].(float64) != 1 || channel["model_count"].(float64) != 0 {
		t.Fatalf("unexpected channel: %+v", channel)
	}

	updated := adminDo(t, handler, http.MethodPut, "/admin/channels/1", map[string]any{
		"name": "OpenAI Updated", "base_url": "https://api2.openai.test", "auth_type": "bearer", "status": 1, "weight": 50, "priority": 20, "balance": "",
	})
	assertNoSecret(t, updated, "sk-secret")
	updatedChannel := updated["data"].(map[string]any)
	if updatedChannel["balance"] != nil {
		t.Fatalf("balance = %v, want null", updatedChannel["balance"])
	}

	status := adminDo(t, handler, http.MethodPut, "/admin/channels/1/status", map[string]any{"status": 0})
	if status["data"].(map[string]any)["status"].(float64) != 0 {
		t.Fatalf("status update failed: %+v", status)
	}

	balance := adminDo(t, handler, http.MethodPut, "/admin/channels/1/balance", map[string]any{"balance": "10.000000", "delta": "2.500000"})
	if balance["data"].(map[string]any)["balance"] != "12.500000" {
		t.Fatalf("balance update failed: %+v", balance)
	}

	listed := adminDo(t, handler, http.MethodGet, "/admin/channels", nil)
	assertNoSecret(t, listed, "sk-secret")
	if listed["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("channel list total mismatch: %+v", listed)
	}
}

func TestChannelCreateRequiresAPIKey(t *testing.T) {
	res := adminRaw(t, newTestHandler(), http.MethodPost, "/admin/channels", map[string]any{"name": "missing key", "base_url": "https://api.test"})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestModelMappingsCatalogPricingAndCascadeDelete(t *testing.T) {
	handler := newTestHandler()
	adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{"name": "OpenAI", "base_url": "https://api.openai.test", "api_key": "sk-secret", "auth_type": "bearer", "status": 1})

	mapping := adminDo(t, handler, http.MethodPost, "/admin/channels/1/models", map[string]any{"model_name": "gpt-4o-mini", "upstream_model": "gpt-4o-mini-up", "enabled": true})
	modelID := int(mapping["data"].(map[string]any)["id"].(float64))

	adminDo(t, handler, http.MethodPut, "/admin/channels/1/models/"+itoa(modelID), map[string]any{"upstream_model": "gpt-4o-mini", "enabled": true})
	catalog := adminDo(t, handler, http.MethodGet, "/admin/models?status=1", nil)
	models := catalog["data"].(map[string]any)["list"].([]any)
	if len(models) != 1 || models[0].(map[string]any)["model_name"] != "gpt-4o-mini" {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	channels := models[0].(map[string]any)["channels"].([]any)
	if len(channels) != 1 || channels[0].(map[string]any)["channel_name"] != "OpenAI" {
		t.Fatalf("unexpected catalog channels: %+v", channels)
	}

	price := adminDo(t, handler, http.MethodPost, "/admin/pricing", map[string]any{"channel_id": 1, "model_name": "gpt-4o-mini", "input_price_per_1m": "0.15000000", "output_price_per_1m": "0.60000000", "currency": "USD"})
	if price["data"].(map[string]any)["upstream_model"] != "gpt-4o-mini" {
		t.Fatalf("pricing upstream model mismatch: %+v", price)
	}

	adminDo(t, handler, http.MethodDelete, "/admin/pricing", map[string]any{"channel_id": 1, "model_name": "gpt-4o-mini"})
	pricingList := adminDo(t, handler, http.MethodGet, "/admin/pricing", nil)
	if pricingList["data"].(map[string]any)["total"].(float64) != 0 {
		t.Fatalf("pricing delete failed: %+v", pricingList)
	}

	adminDo(t, handler, http.MethodPost, "/admin/pricing", map[string]any{"channel_id": 1, "model_name": "gpt-4o-mini", "input_price_per_1m": "0.15000000", "output_price_per_1m": "0.60000000", "currency": "USD"})
	adminDo(t, handler, http.MethodDelete, "/admin/channels/1", nil)
	if adminDo(t, handler, http.MethodGet, "/admin/channels/1/models", nil)["data"].(map[string]any)["total"].(float64) != 0 {
		t.Fatal("model mappings were not cascade deleted")
	}
	if adminDo(t, handler, http.MethodGet, "/admin/pricing", nil)["data"].(map[string]any)["total"].(float64) != 0 {
		t.Fatal("pricing was not cascade deleted")
	}
}

func TestRemoteModelsUsesFakeUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-secret" {
			t.Fatalf("unexpected upstream request path=%s auth=%s", r.URL.Path, r.Header.Get("Authorization"))
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]string{{"id": "gpt-4o-mini"}}})
	}))
	defer upstream.Close()

	handler := newTestHandler()
	adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{"name": "Fake", "base_url": upstream.URL, "api_key": "sk-secret", "auth_type": "bearer", "status": 1})
	remote := adminDo(t, handler, http.MethodPost, "/admin/channels/1/remote-models", map[string]any{})
	data := remote["data"].(map[string]any)
	if data["ok"] != true || len(data["models"].([]any)) != 1 {
		t.Fatalf("unexpected remote models: %+v", remote)
	}
}

func TestInvalidBalanceReturnsBadRequest(t *testing.T) {
	handler := newTestHandler()
	adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{"name": "OpenAI", "base_url": "https://api.openai.test", "api_key": "sk-secret", "auth_type": "bearer", "status": 1})

	res := adminRaw(t, handler, http.MethodPut, "/admin/channels/1/balance", map[string]any{"delta": "abc"})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestDeleteMissingChannelReturnsNotFound(t *testing.T) {
	res := adminRaw(t, newTestHandler(), http.MethodDelete, "/admin/channels/404", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestDeleteMissingModelMappingReturnsNotFound(t *testing.T) {
	handler := newTestHandler()
	adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{"name": "OpenAI", "base_url": "https://api.openai.test", "api_key": "sk-secret", "auth_type": "bearer", "status": 1})

	res := adminRaw(t, handler, http.MethodDelete, "/admin/channels/1/models/404", nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestPricingRejectsInvalidChannelOrModel(t *testing.T) {
	handler := newTestHandler()

	res := adminRaw(t, handler, http.MethodPost, "/admin/pricing", map[string]any{"channel_id": 404, "model_name": "missing", "input_price_per_1m": "0.10000000", "output_price_per_1m": "0.20000000", "currency": "USD"})
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing channel status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}

	adminDo(t, handler, http.MethodPost, "/admin/channels", map[string]any{"name": "OpenAI", "base_url": "https://api.openai.test", "api_key": "sk-secret", "auth_type": "bearer", "status": 1})
	res = adminRaw(t, handler, http.MethodPost, "/admin/pricing", map[string]any{"channel_id": 1, "model_name": "missing", "input_price_per_1m": "0.10000000", "output_price_per_1m": "0.20000000", "currency": "USD"})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("missing model status = %d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func adminDo(t *testing.T, handler http.Handler, method, path string, body any) map[string]any {
	t.Helper()
	res := adminRaw(t, handler, method, path, body)
	if res.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d, want 200; body=%s", method, path, res.Code, res.Body.String())
	}
	var out map[string]any
	decodeJSON(t, res, &out)
	if out["code"].(float64) != 0 {
		t.Fatalf("%s %s code = %v; body=%s", method, path, out["code"], res.Body.String())
	}
	return out
}

func adminRaw(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func assertNoSecret(t *testing.T, value any, secret string) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) || strings.Contains(string(b), "api_key") {
		t.Fatalf("response leaked secret or api_key: %s", b)
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
