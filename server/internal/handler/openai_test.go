package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store/memory"
)

const upstreamKey = "up-secret-key"

func upstreamSuccess() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		model, _ := payload["model"].(string)
		if r.Header.Get("Authorization") != "Bearer "+upstreamKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"model":   model,
			"choices": []map[string]any{{"index": 0, "message": map[string]string{"role": "assistant", "content": "hi"}}},
			"usage":   map[string]any{"prompt_tokens": 1000, "completion_tokens": 500, "total_tokens": 1500},
		})
	})
}

type proxyFixture struct {
	server   *Server
	store    *memory.Store
	fullKey  string
	upstream *httptest.Server
}

func newProxyFixture(t *testing.T, upstream http.Handler) *proxyFixture {
	t.Helper()

	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)

	st := memory.New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000"}); err != nil {
		t.Fatal(err)
	}
	created, err := st.CreateKey(1, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatal(err)
	}

	balance := "10.000000"
	channel, err := st.CreateChannel(domain.ChannelInput{Name: "upstream", BaseURL: server.URL, APIKey: upstreamKey, Status: 1, Priority: 10, Weight: 100, Balance: &balance})
	if err != nil {
		t.Fatal(err)
	}
	channelID := channel["id"].(int)
	if _, err := st.CreateChannelModel(channelID, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up-gpt", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: channelID, ModelName: "gpt", InputPricePer1M: "0.15000000", OutputPricePer1M: "0.60000000", CachedInputPricePer1M: "0.07500000", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}

	return &proxyFixture{
		server:   NewServer(filepath.Join("..", "..", "..", "dashboard"), st),
		store:    st,
		fullKey:  created["full_key"].(string),
		upstream: server,
	}
}

func proxyDo(t *testing.T, f *proxyFixture, method, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res := httptest.NewRecorder()
	f.server.OpenAI(res, req)
	return res
}

func TestOpenAIModels(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())

	ok := proxyDo(t, f, http.MethodGet, "/v1/models", f.fullKey, "")
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ok.Code, ok.Body.String())
	}
	var list domain.OpenAIModelList
	if err := json.Unmarshal(ok.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Object != "list" || len(list.Data) != 1 || list.Data[0].ID != "gpt" {
		t.Fatalf("unexpected model list: %+v", list)
	}

	unauthorized := proxyDo(t, f, http.MethodGet, "/v1/models", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status = %d, want 401", unauthorized.Code)
	}
	invalid := proxyDo(t, f, http.MethodGet, "/v1/models", "sk-invalid", "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid key status = %d, want 401", invalid.Code)
	}
}

func TestChatCompletionsSuccess(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())

	body := `{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`
	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, body)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"model":"gpt"`) {
		t.Fatalf("response model not rewritten: %s", res.Body.String())
	}
	if strings.Contains(res.Body.String(), "up-gpt") {
		t.Fatalf("response leaked upstream model: %s", res.Body.String())
	}
	if strings.Contains(res.Body.String(), upstreamKey) {
		t.Fatalf("response leaked upstream key: %s", res.Body.String())
	}

	balance, err := f.store.GetUserBalance(1)
	if err != nil {
		t.Fatal(err)
	}
	if balance["available_balance"] != "9.999550" {
		t.Fatalf("user balance = %v, want 9.999550", balance["available_balance"])
	}

	channelBalance, err := f.store.GetChannelSecret(1)
	if err != nil {
		t.Fatal(err)
	}
	if channelBalance.Balance == nil || *channelBalance.Balance != "9.999550" {
		t.Fatalf("channel balance = %v, want 9.999550", channelBalance.Balance)
	}

	logs, err := f.store.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if logs.Total != 1 {
		t.Fatalf("usage logs total = %d, want 1", logs.Total)
	}
	entry := logs.List[0].(map[string]any)
	if entry["status"] != "success" || entry["total_tokens"] != 1500 || entry["total_cost"] != "0.000450" {
		t.Fatalf("unexpected usage log: %+v", entry)
	}
	channelID, ok := entry["channel_id"].(*int)
	if !ok || channelID == nil || *channelID != 1 {
		t.Fatalf("usage log channel mismatch: %+v", entry)
	}
	if entry["upstream_model"] != "up-gpt" || entry["model"] != "gpt" {
		t.Fatalf("usage log model mismatch: %+v", entry)
	}
	if entry["unit_price_input_per_1m"] != "0.15000000" || entry["unit_price_output_per_1m"] != "0.60000000" {
		t.Fatalf("usage log unit prices missing: %+v", entry)
	}

	keys, err := f.store.ListUserKeys(1, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if keys.List[0].(map[string]any)["last_used_at"] == nil {
		t.Fatal("last_used_at not updated")
	}
}

func TestChatCompletionsCachedTokensBilling(t *testing.T) {
	f := newProxyFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"id": "chatcmpl-cached", "object": "chat.completion", "model": "up-gpt", "choices": []any{},
			"usage": map[string]any{
				"prompt_tokens": 1000, "completion_tokens": 500, "total_tokens": 1500,
				"prompt_tokens_details": map[string]any{"cached_tokens": 200},
			},
		})
	}))

	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, `{"model":"gpt","messages":[]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", res.Code, res.Body.String())
	}

	// (1000-200)*0.15/1e6 + 200*0.075/1e6 + 500*0.6/1e6 = 0.000435
	balance, _ := f.store.GetUserBalance(1)
	if balance["available_balance"] != "9.999565" {
		t.Fatalf("balance = %v, want 9.999565", balance["available_balance"])
	}
	logs, _ := f.store.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	entry := logs.List[0].(map[string]any)
	if entry["total_cost"] != "0.000435" {
		t.Fatalf("cost = %v, want 0.000435 (cached tokens must not be double charged)", entry["total_cost"])
	}
	if entry["cached_input_tokens"] != 200 {
		t.Fatalf("cached_input_tokens = %v, want 200", entry["cached_input_tokens"])
	}
}

func TestChatCompletionsAuthFailures(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())
	body := `{"model":"gpt","messages":[]}`

	if res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", "", body); res.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status = %d, want 401", res.Code)
	}
	if res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", "sk-bad", body); res.Code != http.StatusUnauthorized {
		t.Fatalf("invalid key status = %d, want 401", res.Code)
	}

	// Disabled key.
	created, err := f.store.CreateKey(1, domain.KeyInput{KeyName: "second"})
	if err != nil {
		t.Fatal(err)
	}
	secondKey := created["full_key"].(string)
	if _, err := f.store.UpdateKey(1, created["id"].(int), domain.KeyUpdateInput{IsActive: boolPointer(false)}); err != nil {
		t.Fatal(err)
	}
	if res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", secondKey, body); res.Code != http.StatusUnauthorized {
		t.Fatalf("disabled key status = %d, want 401", res.Code)
	}

	// Expired key.
	expired, err := f.store.CreateKey(1, domain.KeyInput{KeyName: "expired", ExpiresAt: "2020-01-01T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", expired["full_key"].(string), body); res.Code != http.StatusUnauthorized {
		t.Fatalf("expired key status = %d, want 401", res.Code)
	}

	// Suspended user.
	if _, err := f.store.UpdateUserStatus(1, "suspended"); err != nil {
		t.Fatal(err)
	}
	if res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, body); res.Code != http.StatusForbidden {
		t.Fatalf("suspended user status = %d, want 403", res.Code)
	}
}

func TestChatCompletionsStreamingUnsupported(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())
	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, `{"model":"gpt","stream":true,"messages":[]}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
}

func TestChatCompletionsInsufficientBalance(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())
	// Drain the user balance via a debit.
	if _, err := f.store.DebitUserBalance(1, "10.000000", "drain"); err != nil {
		t.Fatal(err)
	}
	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, `{"model":"gpt","messages":[]}`)
	if res.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body=%s", res.Code, res.Body.String())
	}
}

func TestChatCompletionsNoChannel(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())
	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, `{"model":"unknown-model","messages":[]}`)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.Code)
	}
}

func TestChatCompletionsUpstreamFailureDoesNotCharge(t *testing.T) {
	f := newProxyFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"message": "boom"}})
	}))

	res := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, `{"model":"gpt","messages":[]}`)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want upstream 500 passthrough", res.Code)
	}

	balance, _ := f.store.GetUserBalance(1)
	if balance["available_balance"] != "10.000000" {
		t.Fatalf("balance changed on upstream failure: %v", balance["available_balance"])
	}
	logs, _ := f.store.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if logs.Total != 1 {
		t.Fatalf("usage logs total = %d, want 1 failure log", logs.Total)
	}
	entry := logs.List[0].(map[string]any)
	if entry["status"] != "error" || entry["total_cost"] != "0.000000" {
		t.Fatalf("unexpected failure log: %+v", entry)
	}
}

func TestChatCompletionsRateLimited(t *testing.T) {
	f := newProxyFixture(t, upstreamSuccess())
	rule, err := f.store.CreateRateLimit(domain.RateLimitInput{
		RuleName: stringPointer("global rpm"), TargetType: stringPointer("global"), Metric: stringPointer("rpm"),
		LimitValue: int64Pointer(1), WindowSeconds: intPointer(60), Action: stringPointer("reject"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule["id"] == nil {
		t.Fatal("rule not created")
	}

	body := `{"model":"gpt","messages":[]}`
	first := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, body)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200; body=%s", first.Code, first.Body.String())
	}
	second := proxyDo(t, f, http.MethodPost, "/v1/chat/completions", f.fullKey, body)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429; body=%s", second.Code, second.Body.String())
	}
}

func stringPointer(v string) *string { return &v }
func intPointer(v int) *int          { return &v }
func int64Pointer(v int64) *int64    { return &v }
func boolPointer(v bool) *bool       { return &v }
