package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

func TestPGQuotaReservationRequiresUserAndKeyQuota(t *testing.T) {
	st := testStore(t)
	_, keyID := createQuotaTestIdentity(t, st)
	createQuotaPolicy(t, st, "user daily", "user", 1, "day", 100, "10.000000")
	createQuotaPolicy(t, st, "key daily", "api_key", keyID, "day", 50, "5.000000")

	reservation, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{
		RequestID: "quota-1", UserID: 1, APIKeyID: keyID, Model: "gpt",
		EstimatedTokens: 40, EstimatedCost: "1.000000", ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if reservation.ID == 0 {
		t.Fatal("reservation ID is zero")
	}

	_, err = st.ReserveQuota(context.Background(), domain.QuotaReserveInput{
		RequestID: "quota-2", UserID: 1, APIKeyID: keyID, Model: "gpt",
		EstimatedTokens: 20, EstimatedCost: "1.000000", ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if !errors.Is(err, store.ErrQuotaExceeded) {
		t.Fatalf("second reservation error = %v, want ErrQuotaExceeded", err)
	}

	usage, err := st.ListQuotaUsage(context.Background(), domain.QuotaPolicyFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != 2 {
		t.Fatalf("usage total = %d, want 2", usage.Total)
	}
	for _, item := range usage.List {
		if item.ReservedTokens != 40 || item.UsedTokens != 0 {
			t.Fatalf("partial reservation leaked into bucket: %+v", item)
		}
	}
}

func TestPGQuotaReleaseAndSettlementMoveReservedToUsed(t *testing.T) {
	st := testStore(t)
	_, keyID := createQuotaTestIdentity(t, st)
	createQuotaPolicy(t, st, "user daily", "user", 1, "day", 100, "10.000000")

	released, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{RequestID: "release", UserID: 1, APIKeyID: keyID, Model: "gpt", EstimatedTokens: 40, EstimatedCost: "2.000000", ExpiresAt: time.Now().UTC().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ReleaseQuota(context.Background(), released.ID); err != nil {
		t.Fatal(err)
	}

	settled, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{RequestID: "settle", UserID: 1, APIKeyID: keyID, Model: "gpt", EstimatedTokens: 40, EstimatedCost: "2.000000", ExpiresAt: time.Now().UTC().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	usage := successUsageInput("settle", 1, 0)
	usage.APIKeyID = &keyID
	usage.ChannelID = nil
	usage.TotalTokens = 25
	usage.TotalCost = "1.250000"
	if _, err := st.SettleChatCompletion(domain.ChatSettlementInput{ReservationID: settled.ID, UserID: 1, APIKeyID: keyID, Cost: "1.250000", Description: "chat", UsageLog: usage}); err != nil {
		t.Fatal(err)
	}

	rows, err := st.ListQuotaUsage(context.Background(), domain.QuotaPolicyFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if rows.Total != 1 || rows.List[0].ReservedTokens != 0 || rows.List[0].UsedTokens != 25 || rows.List[0].ReservedCost != "0.000000" || rows.List[0].UsedCost != "1.250000" {
		t.Fatalf("quota usage = %+v", rows)
	}
}

func TestPGQuotaConcurrentReservationsDoNotOversell(t *testing.T) {
	st := testStore(t)
	_, keyID := createQuotaTestIdentity(t, st)
	createQuotaPolicy(t, st, "user daily", "user", 1, "day", 100, "10.000000")

	var successes atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{
				RequestID: fmt.Sprintf("concurrent-%d", i), UserID: 1, APIKeyID: keyID, Model: "gpt",
				EstimatedTokens: 10, EstimatedCost: "0.100000", ExpiresAt: time.Now().UTC().Add(time.Minute),
			})
			if err == nil {
				successes.Add(1)
				return
			}
			if !errors.Is(err, store.ErrQuotaExceeded) {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("unexpected reservation error: %v", err)
	}
	if successes.Load() != 10 {
		t.Fatalf("successful reservations = %d, want 10", successes.Load())
	}
	usage, err := st.ListQuotaUsage(context.Background(), domain.QuotaPolicyFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != 1 || usage.List[0].ReservedTokens != 100 {
		t.Fatalf("quota usage = %+v, want reserved_tokens=100", usage)
	}
}

func TestPGQuotaReaperReleasesExpiredReservation(t *testing.T) {
	st := testStore(t)
	_, keyID := createQuotaTestIdentity(t, st)
	createQuotaPolicy(t, st, "user daily", "user", 1, "day", 100, "10.000000")
	now := time.Now().UTC()
	st.now = func() time.Time { return now }
	reservation, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{
		RequestID: "expired-pg", UserID: 1, APIKeyID: keyID, Model: "gpt",
		EstimatedTokens: 40, EstimatedCost: "1.000000", ExpiresAt: now.Add(time.Minute),
	})
	if err != nil || reservation.ID == 0 {
		t.Fatalf("reserve = %+v, %v", reservation, err)
	}
	now = now.Add(2 * time.Minute)
	count, err := st.ReapExpiredQuotaReservations(context.Background(), 100)
	if err != nil || count != 1 {
		t.Fatalf("reap = %d, %v", count, err)
	}
	usage, err := st.ListQuotaUsage(context.Background(), domain.QuotaPolicyFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if usage.List[0].ReservedTokens != 0 || usage.List[0].ReservedCost != "0.000000" {
		t.Fatalf("quota usage = %+v", usage)
	}
}

func TestPGQuotaMonthlyCostLimitIsEnforced(t *testing.T) {
	st := testStore(t)
	_, keyID := createQuotaTestIdentity(t, st)
	createQuotaPolicy(t, st, "key monthly cost", "api_key", keyID, "month", 100000, "1.000000")
	if _, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{RequestID: "cost-1", UserID: 1, APIKeyID: keyID, EstimatedTokens: 10, EstimatedCost: "0.750000", ExpiresAt: time.Now().UTC().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	_, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{RequestID: "cost-2", UserID: 1, APIKeyID: keyID, EstimatedTokens: 10, EstimatedCost: "0.300000", ExpiresAt: time.Now().UTC().Add(time.Minute)})
	if !errors.Is(err, store.ErrQuotaExceeded) {
		t.Fatalf("error = %v, want ErrQuotaExceeded", err)
	}
}

func createQuotaTestIdentity(t *testing.T, st *Store) (string, int) {
	t.Helper()
	if _, err := st.CreateUser(domain.UserInput{Nickname: "quota-user"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RechargeUser(1, domain.RechargeInput{Amount: "20.000000"}); err != nil {
		t.Fatal(err)
	}
	key, err := st.CreateKey(1, domain.KeyInput{KeyName: "quota-key"})
	if err != nil {
		t.Fatal(err)
	}
	return key.FullKey, key.ID
}

func createQuotaPolicy(t *testing.T, st *Store, name, scope string, scopeID int, period string, tokens int64, cost string) {
	t.Helper()
	if _, err := st.CreateQuotaPolicy(domain.QuotaPolicyInput{PolicyName: &name, ScopeType: &scope, ScopeID: &scopeID, PeriodType: &period, TokenLimit: &tokens, CostLimit: &cost}); err != nil {
		t.Fatal(err)
	}
}
