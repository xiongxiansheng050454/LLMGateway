package httpapi

import (
	"encoding/json"
	"testing"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/usage"
)

func TestAdminDTOJSONContract(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		keys   []string
		absent []string
	}{
		{
			name:   "channel",
			value:  catalog.ChannelDTO{ID: 1, Name: "primary", BaseURL: "https://example.test", AuthType: "bearer", Status: 1, Weight: 10, Priority: 1, Balance: nil, ModelCount: 2},
			keys:   []string{"id", "name", "base_url", "auth_type", "status", "weight", "priority", "balance", "model_count"},
			absent: []string{"api_key", "api_key_ciphertext"},
		},
		{
			name:   "user",
			value:  accounts.UserDTO{ID: 1, Nickname: "Alice", UserGroup: "default", Status: "active", Balance: accounts.BalanceDTO{AvailableBalance: "10.000000", FrozenBalance: "0.000000"}},
			keys:   []string{"id", "nickname", "user_group", "status", "balance"},
			absent: []string{"password", "password_plaintext"},
		},
		{
			name:   "usage log",
			value:  usage.UsageLogDTO{ID: 1, RequestID: "req-1", ChannelName: "primary", Model: "model", TotalTokens: 12, TotalCost: "0.010000", Status: "success", CreatedAt: "2026-01-01T00:00:00Z"},
			keys:   []string{"id", "request_id", "channel_name", "model", "input_tokens", "output_tokens", "cached_input_tokens", "total_tokens", "total_cost", "status", "created_at"},
			absent: []string{},
		},
		{
			name:   "rate limit",
			value:  ratelimit.RateLimitRuleDTO{ID: 1, RuleName: "global-rpm", TargetType: "global", TargetValue: "*", Metric: "rpm", LimitValue: 60, WindowSeconds: 60, Action: "reject", Priority: 100, Enabled: true, Extras: json.RawMessage(`{}`)},
			keys:   []string{"id", "rule_name", "target_type", "target_value", "metric", "limit_value", "window_seconds", "action", "priority", "enabled", "extras"},
			absent: []string{},
		},
		{
			name:   "quota policy",
			value:  quota.QuotaPolicyDTO{ID: 1, PolicyName: "daily", ScopeType: quota.QuotaScopeUser, ScopeID: 1, PeriodType: quota.QuotaPeriodDay, TokenLimit: nil, CostLimit: stringPtr("5.000000"), Enabled: true},
			keys:   []string{"id", "policy_name", "scope_type", "scope_id", "period_type", "token_limit", "cost_limit", "enabled"},
			absent: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]any
			if err := json.Unmarshal(encoded, &object); err != nil {
				t.Fatal(err)
			}
			for _, key := range tt.keys {
				if _, ok := object[key]; !ok {
					t.Errorf("missing JSON field %q in %s", key, encoded)
				}
			}
			for _, key := range tt.absent {
				if _, ok := object[key]; ok {
					t.Errorf("sensitive or unsupported JSON field %q in %s", key, encoded)
				}
			}
		})
	}
}

func stringPtr(value string) *string { return &value }
