package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"LLMGateway/server/internal/store"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func strPtr(value string) *string {
	return &value
}

func TestPGChannelCRUDAndSecretEncryption(t *testing.T) {
	st := testStore(t)

	created, err := st.CreateChannel(domain.ChannelInput{
		Name: "OpenAI", BaseURL: "https://api.openai.test", APIKey: "sk-plaintext-secret", AuthType: "bearer", Status: 1, Weight: 100, Priority: 10, Balance: strPtr("100.000000"),
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if created.ID != 1 || created.Name != "OpenAI" || created.ModelCount != 0 {
		t.Fatalf("unexpected created channel: %+v", created)
	}
	if created.Balance == nil || *created.Balance != "100.000000" {
		t.Fatalf("unexpected balance: %+v", created.Balance)
	}

	// api_key must be stored as ciphertext, not plaintext.
	var ciphertext string
	if err := st.pool.QueryRow(context.Background(), "SELECT api_key_ciphertext FROM channels WHERE id = $1", 1).Scan(&ciphertext); err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	if ciphertext == "sk-plaintext-secret" || ciphertext == "" {
		t.Fatalf("api_key stored in plaintext: %q", ciphertext)
	}
	if _, err := st.cipher.Decrypt(ciphertext); err != nil {
		t.Fatalf("ciphertext not decryptable: %v", err)
	}

	secret, err := st.GetChannelSecret(1)
	if err != nil {
		t.Fatalf("GetChannelSecret: %v", err)
	}
	if secret.APIKey != "sk-plaintext-secret" {
		t.Fatalf("GetChannelSecret api_key = %q", secret.APIKey)
	}

	// Updating without api_key must keep the existing ciphertext.
	if _, err := st.UpdateChannel(1, domain.ChannelInput{Name: "OpenAI Updated", BaseURL: "https://api2.test", AuthType: "bearer", Status: 1, Weight: 50, Priority: 20, Balance: strPtr("")}); err != nil {
		t.Fatalf("UpdateChannel: %v", err)
	}
	var afterUpdate string
	if err := st.pool.QueryRow(context.Background(), "SELECT api_key_ciphertext FROM channels WHERE id = $1", 1).Scan(&afterUpdate); err != nil {
		t.Fatal(err)
	}
	if afterUpdate != ciphertext {
		t.Fatalf("api_key changed on update without api_key")
	}
	updated, _ := st.GetChannelSecret(1)
	if updated.APIKey != "sk-plaintext-secret" {
		t.Fatalf("api_key lost after update: %q", updated.APIKey)
	}
	if updated.Balance != nil {
		t.Fatalf("empty balance should clear to nil, got %v", *updated.Balance)
	}

	// Updating with api_key rotates the ciphertext.
	if _, err := st.UpdateChannel(1, domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", AuthType: "bearer", Status: 1, Weight: 100, Priority: 10, APIKey: "sk-rotated", Balance: strPtr("5.000000")}); err != nil {
		t.Fatalf("UpdateChannel rotate: %v", err)
	}
	rotated, _ := st.GetChannelSecret(1)
	if rotated.APIKey != "sk-rotated" {
		t.Fatalf("api_key not rotated: %q", rotated.APIKey)
	}

	statusDTO, err := st.UpdateChannelStatus(1, 0)
	if err != nil {
		t.Fatalf("UpdateChannelStatus: %v", err)
	}
	if statusDTO.Status != 0 {
		t.Fatalf("status = %v, want 0", statusDTO.Status)
	}

	listed, err := st.ListChannels()
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if listed.Total != 1 {
		t.Fatalf("ListChannels total = %d, want 1", listed.Total)
	}

	if err := st.DeleteChannel(1); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}
	if err := st.DeleteChannel(1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteChannel err = %v, want ErrNotFound", err)
	}
}

func TestPGChannelBalanceMathAndInvalidInput(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk-secret", Status: 1, Balance: strPtr("10.000000")}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	dto, err := st.UpdateChannelBalance(1, "", "2.500000")
	if err != nil {
		t.Fatalf("UpdateChannelBalance delta: %v", err)
	}
	if dto.Balance == nil || *dto.Balance != "12.500000" {
		t.Fatalf("balance after delta = %+v, want 12.500000", dto.Balance)
	}

	dto, err = st.UpdateChannelBalance(1, "1.000000", "-0.250000")
	if err != nil {
		t.Fatalf("UpdateChannelBalance set+delta: %v", err)
	}
	if *dto.Balance != "0.750000" {
		t.Fatalf("balance = %+v, want 0.750000", dto.Balance)
	}

	if _, err := st.UpdateChannelBalance(1, "", "abc"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid delta err = %v, want ErrInvalid", err)
	}
	if _, err := st.UpdateChannelBalance(1, "", ""); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("empty balance/delta err = %v, want ErrInvalid", err)
	}
	if _, err := st.UpdateChannelBalance(404, "", "1.000000"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing channel err = %v, want ErrNotFound", err)
	}
}

func TestPGModelMappingsCatalogAndCascade(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk-secret", Status: 1}); err != nil {
		t.Fatal(err)
	}

	mapping, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "gpt-4o-mini-up", Enabled: true})
	if err != nil {
		t.Fatalf("CreateChannelModel: %v", err)
	}
	if mapping.ID == 0 || mapping.ModelName != "gpt-4o-mini" {
		t.Fatalf("unexpected mapping: %+v", mapping)
	}

	if _, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "dup", Enabled: true}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("duplicate mapping err = %v, want ErrInvalid", err)
	}
	if _, err := st.CreateChannelModel(404, domain.ChannelModel{ModelName: "x", UpstreamModel: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing channel err = %v, want ErrNotFound", err)
	}

	if _, err := st.UpdateChannelModel(1, mapping.ID, "gpt-4o-mini", true); err != nil {
		t.Fatalf("UpdateChannelModel: %v", err)
	}
	if _, err := st.UpdateChannelModel(1, 404, "x", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing mapping err = %v, want ErrNotFound", err)
	}

	catalog, err := st.ListCatalogModels(true)
	if err != nil {
		t.Fatalf("ListCatalogModels: %v", err)
	}
	if catalog.Total != 1 {
		t.Fatalf("catalog total = %d, want 1", catalog.Total)
	}
	entry := catalog.List[0]
	if entry.ModelName != "gpt-4o-mini" || entry.Status != 1 {
		t.Fatalf("unexpected catalog entry: %+v", entry)
	}
	channels := entry.Channels
	if len(channels) != 1 || channels[0].ChannelName != "OpenAI" {
		t.Fatalf("unexpected catalog channels: %+v", channels)
	}

	// Pricing then cascade delete via channel.
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.15000000", OutputPricePer1M: "0.60000000", Currency: "USD"}); err != nil {
		t.Fatalf("UpsertPricing: %v", err)
	}
	if err := st.DeleteChannel(1); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}
	if models, _ := st.ListChannelModels(1); models.Total != 0 {
		t.Fatalf("mappings not cascade deleted: %d", models.Total)
	}
	if pricing, _ := st.ListPricing(); pricing.Total != 0 {
		t.Fatalf("pricing not cascade deleted: %d", pricing.Total)
	}
}

func TestPGPricingUpsertValidationAndDelete(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk-secret", Status: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 404, ModelName: "x", InputPricePer1M: "0.10000000", OutputPricePer1M: "0.20000000"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing channel err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "missing", InputPricePer1M: "0.10000000", OutputPricePer1M: "0.20000000"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("missing mapping err = %v, want ErrInvalid", err)
	}

	if _, err := st.CreateChannelModel(1, domain.ChannelModel{ModelName: "gpt-4o-mini", UpstreamModel: "gpt-4o-mini-up", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	dto, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.15000000", OutputPricePer1M: "0.60000000", CachedInputPricePer1M: "0.07500000", Currency: "USD"})
	if err != nil {
		t.Fatalf("UpsertPricing: %v", err)
	}
	if dto.ChannelName != "OpenAI" || dto.UpstreamModel != "gpt-4o-mini-up" {
		t.Fatalf("unexpected pricing dto: %+v", dto)
	}
	if dto.InputPricePer1M != "0.15000000" || dto.CachedInputPricePer1M != "0.07500000" {
		t.Fatalf("unexpected price format: %+v", dto)
	}

	// Overwrite (upsert) the same channel+model.
	overwritten, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "0.20000000", OutputPricePer1M: "0.70000000", Currency: "USD"})
	if err != nil {
		t.Fatalf("UpsertPricing overwrite: %v", err)
	}
	if overwritten.InputPricePer1M != "0.20000000" {
		t.Fatalf("upsert did not overwrite: %+v", overwritten)
	}

	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: 1, ModelName: "gpt-4o-mini", InputPricePer1M: "bad", OutputPricePer1M: "0.20000000"}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid price err = %v, want ErrInvalid", err)
	}

	if err := st.DeletePricing(domain.DeletePricingInput{ChannelID: 1, ModelName: "gpt-4o-mini"}); err != nil {
		t.Fatalf("DeletePricing: %v", err)
	}
	if pricing, _ := st.ListPricing(); pricing.Total != 0 {
		t.Fatalf("pricing not deleted: %d", pricing.Total)
	}
}

func TestPGChannelBalanceConcurrentDeltas(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk-secret", Status: 1, Balance: strPtr("0.000000")}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.UpdateChannelBalance(1, "", "1.000000"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent UpdateChannelBalance: %v", err)
	}

	secret, err := st.GetChannelSecret(1)
	if err != nil {
		t.Fatalf("GetChannelSecret: %v", err)
	}
	if secret.Balance == nil || *secret.Balance != "20.000000" {
		t.Fatalf("balance after %d concurrent deltas = %v, want 20.000000", workers, secret.Balance)
	}
}

func TestPGMissingResourcesReturnNotFound(t *testing.T) {
	st := testStore(t)

	if _, err := st.UpdateChannel(404, domain.ChannelInput{Name: "x"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("UpdateChannel err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpdateChannelStatus(404, 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("UpdateChannelStatus err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetChannelSecret(404); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetChannelSecret err = %v, want ErrNotFound", err)
	}
	if err := st.DeleteChannelModel(1, 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("DeleteChannelModel err = %v, want ErrNotFound", err)
	}
}
