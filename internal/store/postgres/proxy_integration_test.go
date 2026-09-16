package postgres

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"LLMGateway/internal/crypto"
	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
	"LLMGateway/internal/store/memory"
)

func createKeyAndHash(t *testing.T, st store.Store, userID int) (int, string) {
	t.Helper()
	created, err := st.CreateKey(userID, domain.KeyInput{KeyName: "default", Prefix: "sk-"})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	return created["id"].(int), crypto.HashKey(created["full_key"].(string))
}

func TestPGProxyStoreCapabilities(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "25.000000"}); err != nil {
		t.Fatal(err)
	}
	keyID, keyHash := createKeyAndHash(t, st, 1)

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
	if err := st.UpdateKeyLastUsed(404); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing key err = %v, want ErrNotFound", err)
	}

	result, err := st.DebitUserBalance(1, "3.5", "request")
	if err != nil {
		t.Fatalf("DebitUserBalance: %v", err)
	}
	if result["balance_after"] != "21.500000" {
		t.Fatalf("balance_after = %v, want 21.500000", result["balance_after"])
	}
	if _, err := st.DebitUserBalance(1, "100.000000", "too much"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("insufficient err = %v, want ErrInvalid", err)
	}
	if _, err := st.DebitUserBalance(404, "1.000000", "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}

	// Pricing + route candidates.
	create := func(name string, status, priority, weight int, balance string) int {
		created, err := st.CreateChannel(domain.ChannelInput{Name: name, BaseURL: "https://" + name + ".test", APIKey: "sk", Status: status, Priority: priority, Weight: weight, Balance: strPtr(balance)})
		if err != nil {
			t.Fatal(err)
		}
		return created["id"].(int)
	}
	channelA := create("A", 1, 10, 100, "5.000000")
	channelB := create("B", 1, 10, 200, "")
	channelC := create("C", 1, 5, 100, "")
	disabledChannel := create("D", 0, 99, 999, "")
	for _, id := range []int{channelA, channelB, channelC, disabledChannel} {
		if _, err := st.CreateChannelModel(id, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up-gpt", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: channelA, ModelName: "gpt", InputPricePer1M: "0.10000000", OutputPricePer1M: "0.20000000", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	pricing, err := st.GetPricing(channelA, "gpt")
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}
	if pricing["input_price_per_1m"] != "0.10000000" || pricing["upstream_model"] != "up-gpt" {
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
	if candidates.List[0].(map[string]any)["channel_id"] != channelB {
		t.Fatalf("first candidate = %+v, want channel B", candidates.List[0])
	}
	if candidates.List[1].(map[string]any)["channel_id"] != channelA {
		t.Fatalf("second candidate = %+v, want channel A", candidates.List[1])
	}
	if candidates.List[2].(map[string]any)["channel_id"] != channelC {
		t.Fatalf("third candidate = %+v, want channel C", candidates.List[2])
	}

	// Window counting counts all attempts, including failures.
	if _, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-1", UserID: intp(1), APIKeyID: intp(keyID), ChannelID: intp(channelA), Model: "gpt", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-2", UserID: intp(1), APIKeyID: intp(keyID), ChannelID: intp(channelA), Model: "gpt", Status: "error"}); err != nil {
		t.Fatal(err)
	}
	count, err := st.CountRequestsSince(1, nil, "1970-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("CountRequestsSince: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	scoped, err := st.CountRequestsSince(1, intp(keyID), "1970-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if scoped != 2 {
		t.Fatalf("key-scoped count = %d, want 2", scoped)
	}
	if _, err := st.CountRequestsSince(1, nil, "abc"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid since err = %v, want ErrInvalid", err)
	}
	if _, err := st.CountRequestsSince(1, nil, ""); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("empty since err = %v, want ErrInvalid", err)
	}
}

func TestPGDebitUserBalanceConcurrent(t *testing.T) {
	st := testStore(t)

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

	balance, err := st.GetUserBalance(1)
	if err != nil {
		t.Fatal(err)
	}
	if balance["available_balance"] != "80.000000" {
		t.Fatalf("balance = %v, want 80.000000", balance["available_balance"])
	}
}

// proxySnapshot captures the fields that must match between memory and PG.
type proxySnapshot struct {
	authActive       bool
	authUserStatus   string
	authBalance      string
	authPermissions  string
	pricingInput     string
	pricingUpstream  string
	pricingCached    string
	pricingCurrency  string
	routeOrder       []int
	routeUpstream    string
	routeBalance     string
	afterDebit       string
	insufficientErr  string
	missingPricingIs bool
	countAll         int
}

func runProxyScenario(t *testing.T, st store.Store) proxySnapshot {
	t.Helper()

	if _, err := st.CreateUser(domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "25.000000"}); err != nil {
		t.Fatal(err)
	}
	_, keyHash := createKeyAndHash(t, st, 1)

	auth, err := st.AuthenticateKey(keyHash)
	if err != nil {
		t.Fatal(err)
	}

	create := func(name string, priority, weight int, balance string) int {
		var balancePtr *string
		if balance != "" {
			balancePtr = &balance
		}
		created, err := st.CreateChannel(domain.ChannelInput{Name: name, BaseURL: "https://" + name + ".test", APIKey: "sk", Status: 1, Priority: priority, Weight: weight, Balance: balancePtr})
		if err != nil {
			t.Fatal(err)
		}
		return created["id"].(int)
	}
	channelA := create("A", 10, 100, "5.000000")
	channelB := create("B", 10, 200, "")
	for _, id := range []int{channelA, channelB} {
		if _, err := st.CreateChannelModel(id, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up-gpt", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.UpsertPricing(domain.PricingInput{ChannelID: channelA, ModelName: "gpt", InputPricePer1M: "0.10000000", OutputPricePer1M: "0.20000000", CachedInputPricePer1M: "0.05000000", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}

	pricing, err := st.GetPricing(channelA, "gpt")
	if err != nil {
		t.Fatal(err)
	}
	_, missingPricingErr := st.GetPricing(channelA, "missing")

	candidates, err := st.RouteCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	routeOrder := []int{}
	routeUpstream := ""
	routeBalance := ""
	for _, item := range candidates.List {
		entry := item.(map[string]any)
		routeOrder = append(routeOrder, entry["channel_id"].(int))
		routeUpstream = entry["upstream_model"].(string)
		if balance, ok := entry["balance"].(*string); ok && balance != nil {
			routeBalance = *balance
		}
	}

	debit, err := st.DebitUserBalance(1, "3.5", "request")
	if err != nil {
		t.Fatal(err)
	}
	_, insufficientErr := st.DebitUserBalance(1, "100.000000", "too much")

	if _, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-1", UserID: intp(1), Model: "gpt", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	countAll, err := st.CountRequestsSince(1, nil, "1970-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	return proxySnapshot{
		authActive:       auth.KeyActive,
		authUserStatus:   auth.UserStatus,
		authBalance:      auth.AvailableBalance,
		authPermissions:  string(auth.Permissions),
		pricingInput:     pricing["input_price_per_1m"].(string),
		pricingUpstream:  pricing["upstream_model"].(string),
		pricingCached:    pricing["cached_input_price_per_1m"].(string),
		pricingCurrency:  pricing["currency"].(string),
		routeOrder:       routeOrder,
		routeUpstream:    routeUpstream,
		routeBalance:     routeBalance,
		afterDebit:       debit["balance_after"].(string),
		insufficientErr:  errorKind(insufficientErr),
		missingPricingIs: errors.Is(missingPricingErr, store.ErrNotFound),
		countAll:         countAll,
	}
}

func errorKind(err error) string {
	switch {
	case err == nil:
		return "nil"
	case errors.Is(err, store.ErrInvalid):
		return "invalid"
	case errors.Is(err, store.ErrNotFound):
		return "notfound"
	default:
		return "other"
	}
}

func TestMemoryVsPGProxyConsistency(t *testing.T) {
	pg := runProxyScenario(t, testStore(t))
	mem := runProxyScenario(t, memory.New())

	if !reflect.DeepEqual(pg, mem) {
		t.Fatalf("memory and PG differ:\npg  = %+v\nmem = %+v", pg, mem)
	}
}
