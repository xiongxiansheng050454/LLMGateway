package memory

import (
	"errors"
	"testing"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
)

func strPtr(value string) *string {
	return &value
}

func createTestChannel(t *testing.T, st *Store, balance *string) map[string]any {
	t.Helper()
	created, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk-secret", Status: 1, Balance: balance})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	return created
}

func TestCreateChannelNormalizesBalance(t *testing.T) {
	st := New()

	created := createTestChannel(t, st, strPtr("10.5"))
	if got := *(created["balance"].(*string)); got != "10.500000" {
		t.Fatalf("balance = %q, want 10.500000", got)
	}

	empty := createTestChannel(t, st, strPtr(""))
	if balance, ok := empty["balance"].(*string); !ok || balance != nil {
		t.Fatalf("empty balance should be nil, got %v", empty["balance"])
	}

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "bad", BaseURL: "https://api.test", APIKey: "sk-secret", Balance: strPtr("abc")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid balance err = %v, want ErrInvalid", err)
	}
}

func TestUpdateChannelNormalizesBalance(t *testing.T) {
	st := New()
	createTestChannel(t, st, strPtr("1.000000"))

	updated, err := st.UpdateChannel(1, domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", Balance: strPtr("10.5")})
	if err != nil {
		t.Fatalf("UpdateChannel: %v", err)
	}
	if got := *(updated["balance"].(*string)); got != "10.500000" {
		t.Fatalf("balance = %q, want 10.500000", got)
	}

	if _, err := st.UpdateChannel(1, domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", Balance: strPtr("bad")}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid balance err = %v, want ErrInvalid", err)
	}
}

func TestCreateChannelModelRejectsDuplicate(t *testing.T) {
	st := New()
	createTestChannel(t, st, nil)

	if _, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "gpt-4o-mini-up", Enabled: true}); err != nil {
		t.Fatalf("first mapping: %v", err)
	}
	if _, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "other", Enabled: true}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("duplicate mapping err = %v, want ErrInvalid", err)
	}
}

func TestUpsertPricingNormalizesAndValidates(t *testing.T) {
	st := New()
	createTestChannel(t, st, nil)
	if _, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "gpt-4o-mini-up", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	dto, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.1", OutputPricePer1M: "0.2", Currency: "USD"})
	if err != nil {
		t.Fatalf("UpsertPricing: %v", err)
	}
	if dto["input_price_per_1m"] != "0.10000000" || dto["output_price_per_1m"] != "0.20000000" {
		t.Fatalf("prices not normalized: %+v", dto)
	}

	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "bad", OutputPricePer1M: "0.2"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid input price err = %v, want ErrInvalid", err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.1"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("missing output price err = %v, want ErrInvalid", err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.1", OutputPricePer1M: "0.2", CachedInputPricePer1M: "bad"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid cached price err = %v, want ErrInvalid", err)
	}
}
