package memory

import (
	"errors"
	"sync"
	"testing"

	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

func seedKey(t *testing.T, st *Store, userID int) (keyID int, keyHash string) {
	t.Helper()
	created, err := st.CreateKey(userID, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	fullKey := created.FullKey
	return created.ID, crypto.HashKey(fullKey)
}

func TestAuthenticateKey(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "25.000000"}); err != nil {
		t.Fatal(err)
	}
	keyID, keyHash := seedKey(t, st, 1)

	auth, err := st.AuthenticateKey(keyHash)
	if err != nil {
		t.Fatalf("AuthenticateKey: %v", err)
	}
	if auth.KeyID != keyID || auth.UserID != 1 || !auth.KeyActive || auth.UserStatus != "active" {
		t.Fatalf("unexpected auth context: %+v", auth)
	}
	if auth.AvailableBalance != "25.000000" || auth.FrozenBalance != "0.000000" {
		t.Fatalf("unexpected balances: %+v", auth)
	}
	if string(auth.Permissions) != `{"models":["*"]}` {
		t.Fatalf("permissions = %s", auth.Permissions)
	}

	if _, err := st.AuthenticateKey(crypto.HashKey("sk-unknown")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown key err = %v, want ErrNotFound", err)
	}

	if err := st.UpdateKeyLastUsed(keyID); err != nil {
		t.Fatalf("UpdateKeyLastUsed: %v", err)
	}
	if st.keys[keyID].lastUsedAt == nil {
		t.Fatal("last_used_at not set")
	}
	if err := st.UpdateKeyLastUsed(404); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing key err = %v, want ErrNotFound", err)
	}
}

func TestDebitUserBalance(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "10.000000"}); err != nil {
		t.Fatal(err)
	}

	result, err := st.DebitUserBalance(1, "3.5", "request")
	if err != nil {
		t.Fatalf("DebitUserBalance: %v", err)
	}
	if result.BalanceAfter != "6.500000" {
		t.Fatalf("balance_after = %v, want 6.500000", result.BalanceAfter)
	}

	if _, err := st.DebitUserBalance(1, "100.000000", "too much"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("insufficient err = %v, want ErrInvalid", err)
	}
	if _, err := st.DebitUserBalance(1, "-1.000000", "bad"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("negative err = %v, want ErrInvalid", err)
	}
	if _, err := st.DebitUserBalance(404, "1.000000", "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}

	// Balance must remain unchanged after the rejected debits.
	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "6.500000" {
		t.Fatalf("balance changed by rejected debit: %v", balance.AvailableBalance)
	}
}

func TestDebitUserBalanceConcurrent(t *testing.T) {
	st := New()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "100.000000"}); err != nil {
		t.Fatal(err)
	}

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.DebitUserBalance(1, "1.000000", "concurrent"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent debit: %v", err)
	}

	balance, _ := st.GetUserBalance(1)
	if balance.AvailableBalance != "80.000000" {
		t.Fatalf("balance = %v, want 80.000000", balance.AvailableBalance)
	}
}

func TestGetPricingAndRouteCandidates(t *testing.T) {
	st := New()
	create := func(name string, status, priority, weight int, balance string) int {
		created, err := st.CreateChannel(domain.ChannelInput{Name: name, BaseURL: "https://" + name + ".test", APIKey: "sk", Status: status, Priority: priority, Weight: weight, Balance: strPtr(balance)})
		if err != nil {
			t.Fatal(err)
		}
		return created.ID
	}
	channelA := create("A", 1, 10, 100, "5.000000")
	channelB := create("B", 1, 10, 200, "")
	channelC := create("C", 1, 5, 100, "")
	channelD := create("D", 0, 99, 999, "")

	for _, id := range []int{channelA, channelB, channelC, channelD} {
		if _, err := st.CreateChannelModel(id, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up-gpt", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	// Disabled mapping on an enabled channel must be excluded.
	disabled := create("E", 1, 50, 50, "")
	if _, err := st.CreateChannelModel(disabled, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up-gpt", Enabled: false}); err != nil {
		t.Fatal(err)
	}

	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: channelA, ModelName: "gpt", InputPricePer1M: "0.10000000", OutputPricePer1M: "0.20000000", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	pricing, err := st.GetPricing(channelA, "gpt")
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}
	if pricing.InputPricePer1M != "0.10000000" || pricing.UpstreamModel != "up-gpt" {
		t.Fatalf("unexpected pricing: %+v", pricing)
	}
	if _, err := st.GetPricing(channelA, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing pricing err = %v, want ErrNotFound", err)
	}

	candidates, err := st.RouteCandidates("gpt")
	if err != nil {
		t.Fatalf("RouteCandidates: %v", err)
	}
	if candidates.Total != 3 {
		t.Fatalf("candidates total = %d, want 3", candidates.Total)
	}
	first := candidates.List[0]
	if first.ChannelID != channelB || first.Weight != 200 {
		t.Fatalf("first candidate = %+v, want channel B (weight 200)", first)
	}
	second := candidates.List[1]
	if second.ChannelID != channelA {
		t.Fatalf("second candidate = %+v, want channel A", second)
	}
	third := candidates.List[2]
	if third.ChannelID != channelC {
		t.Fatalf("third candidate = %+v, want channel C", third)
	}
	if third.UpstreamModel != "up-gpt" {
		t.Fatalf("candidate missing upstream_model: %+v", third)
	}
}

func TestCountRequestsSince(t *testing.T) {
	st := New()
	seedLog(st, intp(1), intp(1), "gpt", "success", "0.001000", 10, "2026-09-16T10:00:00Z")
	seedLog(st, intp(1), intp(1), "gpt", "error", "0.001000", 10, "2026-09-16T10:00:30Z")
	seedLog(st, intp(2), intp(1), "gpt", "success", "0.001000", 10, "2026-09-16T10:00:45Z")
	st.usageLogs[0].APIKeyID = intp(1)
	st.usageLogs[1].APIKeyID = intp(1)

	count, err := st.CountRequestsSince(1, nil, "2026-09-16T10:00:15Z")
	if err != nil {
		t.Fatalf("CountRequestsSince: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1 (all attempts after window start)", count)
	}

	count, err = st.CountRequestsSince(1, nil, "2026-09-16T09:59:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	apiKeyID := 1
	count, err = st.CountRequestsSince(1, &apiKeyID, "2026-09-16T09:59:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("key-scoped count = %d, want 2", count)
	}

	if _, err := st.CountRequestsSince(1, nil, "abc"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid since err = %v, want ErrInvalid", err)
	}
	if _, err := st.CountRequestsSince(1, nil, ""); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("empty since err = %v, want ErrInvalid", err)
	}
}
