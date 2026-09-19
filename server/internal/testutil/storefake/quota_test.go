package storefake

import (
	"context"
	"testing"
	"time"

	domain "LLMGateway/server/internal/testutil/testtypes"
)

func TestQuotaReaperReleasesExpiredReservation(t *testing.T) {
	now := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	st := NewWithClock(func() time.Time { return now })
	if _, err := st.CreateUser(domain.UserInput{Nickname: "quota"}); err != nil {
		t.Fatal(err)
	}
	key, err := st.CreateKey(1, domain.KeyInput{KeyName: "quota"})
	if err != nil {
		t.Fatal(err)
	}
	name, scope, period, scopeID, limit := "daily", "user", "day", 1, int64(100)
	if _, err := st.CreateQuotaPolicy(domain.QuotaPolicyInput{PolicyName: &name, ScopeType: &scope, ScopeID: &scopeID, PeriodType: &period, TokenLimit: &limit}); err != nil {
		t.Fatal(err)
	}
	reservation, err := st.ReserveQuota(context.Background(), domain.QuotaReserveInput{RequestID: "expired", UserID: 1, APIKeyID: key.ID, EstimatedTokens: 40, EstimatedCost: "0.000000", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if reservation.ID == 0 {
		t.Fatal("reservation not created")
	}
	now = now.Add(2 * time.Minute)
	count, err := st.ReapExpiredQuotaReservations(context.Background(), 100)
	if err != nil || count != 1 {
		t.Fatalf("reap = %d, %v", count, err)
	}
	usage, _ := st.ListQuotaUsage(context.Background(), domain.QuotaPolicyFilter{Page: 1, PageSize: 10})
	if usage.List[0].ReservedTokens != 0 {
		t.Fatalf("usage = %+v", usage)
	}
}
