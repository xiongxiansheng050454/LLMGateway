package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/testutil/storefake"
)

func newProtocolSeamService(t *testing.T, transport http.RoundTripper, adapter ProtocolAdapter) (*Service, *storefake.Store, *domain.AuthContext) {
	t.Helper()
	st := storefake.New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "seam-user"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000"}); err != nil {
		t.Fatal(err)
	}
	createdKey, err := st.CreateKey(1, domain.KeyInput{KeyName: "seam-key", Prefix: "sk-"})
	if err != nil {
		t.Fatal(err)
	}
	balance := "10.000000"
	channel, err := st.CreateChannel(domain.ChannelInput{
		Name: "seam-upstream", BaseURL: "http://upstream.test", APIKey: "secret", AuthType: "bearer",
		Status: 1, Priority: 1, Weight: 1, Balance: &balance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateChannelModel(channel.ID, domain.ChannelModel{ModelName: "public-model", UpstreamModel: "configured-model", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: channel.ID, ModelName: "public-model", InputPricePer1M: "0.15000000", OutputPricePer1M: "0.60000000", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}

	service := NewService(st, &http.Client{Transport: transport}, func(int) int { return 0 }, time.Now, adapter)
	auth := &domain.AuthContext{KeyID: createdKey.ID, UserID: 1, KeyActive: true, UserStatus: "active", AvailableBalance: "10.000000"}
	return service, st, auth
}

func TestChatCompletionsUsesProtocolNeutralAdapterAndSettles(t *testing.T) {
	var upstreamBody string
	adapter := ProtocolAdapter{
		RewriteRequest: func(body []byte, model string) ([]byte, error) {
			if string(body) != `{"model":"public-model"}` || model != "configured-model" {
				t.Fatalf("rewrite input = %s, model = %q", body, model)
			}
			return []byte(`{"model":"configured-model"}`), nil
		},
		ParseUsage: func(body []byte) *Usage {
			if string(body) != `{"upstream":true}` {
				t.Fatalf("parse body = %s", body)
			}
			return &Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}
		},
		RewriteResponse: func(body []byte, model string) []byte {
			if string(body) != `{"upstream":true}` || model != "public-model" {
				t.Fatalf("response rewrite input = %s, model = %q", body, model)
			}
			return []byte(`{"model":"public-model"}`)
		},
		EstimateUsage: func([]byte, int) (EstimatedUsage, error) {
			return EstimatedUsage{InputTokens: 100, OutputTokens: 100, TotalTokens: 200}, nil
		},
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		upstreamBody = string(body)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"upstream":true}`)), Header: make(http.Header)}, nil
	})
	service, st, auth := newProtocolSeamService(t, transport, adapter)
	response, err := service.ChatCompletions(context.Background(), auth, ChatRequest{Model: "public-model", Body: []byte(`{"model":"public-model"}`)}, "127.0.0.1")
	if err != nil {
		t.Fatalf("ChatCompletions: %v", err)
	}
	if upstreamBody != `{"model":"configured-model"}` {
		t.Fatalf("upstream body = %s", upstreamBody)
	}
	if response.Status != http.StatusOK || string(response.Body) != `{"model":"public-model"}` {
		t.Fatalf("response = %+v", response)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 1500 {
		t.Fatalf("response usage = %+v", response.Usage)
	}
	logs, err := st.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if logs.Total != 1 || len(logs.List) != 1 || logs.List[0].Status != "success" {
		t.Fatalf("usage logs = %+v", logs)
	}
}

func TestChatCompletionsPassesThroughUpstreamErrors(t *testing.T) {
	adapter := ProtocolAdapter{
		RewriteRequest: func(body []byte, model string) ([]byte, error) { return body, nil },
		ParseUsage:     func([]byte) *Usage { t.Fatal("ParseUsage called for upstream error"); return nil },
		RewriteResponse: func([]byte, string) []byte {
			t.Fatal("RewriteResponse called for upstream error")
			return nil
		},
		EstimateUsage: func([]byte, int) (EstimatedUsage, error) {
			return EstimatedUsage{InputTokens: 100, OutputTokens: 100, TotalTokens: 200}, nil
		},
	}
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(`{"error":"busy"}`)), Header: make(http.Header)}, nil
	})
	service, _, auth := newProtocolSeamService(t, transport, adapter)
	response, err := service.ChatCompletions(context.Background(), auth, ChatRequest{Model: "public-model", Body: []byte(`{"model":"public-model"}`)}, "127.0.0.1")
	if err != nil {
		t.Fatalf("ChatCompletions: %v", err)
	}
	if response.Status != http.StatusTooManyRequests || string(response.Body) != `{"error":"busy"}` {
		t.Fatalf("response = %+v", response)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
