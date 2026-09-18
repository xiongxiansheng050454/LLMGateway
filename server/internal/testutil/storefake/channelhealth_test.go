package storefake

import (
	"context"
	"sync"
	"testing"
	"time"

	"LLMGateway/server/internal/domain"
)

func newHealthTestStore() (*Store, *time.Time) {
	current := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	st := NewWithClock(func() time.Time { return current })
	return st, &current
}

func TestChannelHealthLifecycle(t *testing.T) {
	st, clock := newHealthTestStore()

	// Missing row is closed.
	health, err := st.GetChannelHealth(1)
	if err != nil {
		t.Fatal(err)
	}
	if health.State != domain.HealthClosed {
		t.Fatalf("initial state = %s, want closed", health.State)
	}

	// Five consecutive failures trip the breaker.
	for i := 0; i < 5; i++ {
		health, err = st.RecordChannelFailure(1, domain.FailureUpstream5xx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if health.State != domain.HealthOpen || health.OpenedAt == nil {
		t.Fatalf("state = %s, want open with opened_at", health.State)
	}
	if health.FailureCount != 5 {
		t.Fatalf("failure_count = %d, want 5", health.FailureCount)
	}

	// Before the cooldown the channel is still open.
	if got, _ := st.GetChannelHealth(1); got.State != domain.HealthOpen {
		t.Fatalf("before cooldown state = %s, want open", got.State)
	}

	// After the cooldown it becomes half-open lazily.
	*clock = clock.Add(30 * time.Second)
	if got, _ := st.GetChannelHealth(1); got.State != domain.HealthHalfOpen {
		t.Fatalf("after cooldown state = %s, want half-open", got.State)
	}

	// A success closes it again.
	closed, err := st.RecordChannelSuccess(1)
	if err != nil {
		t.Fatal(err)
	}
	if closed.State != domain.HealthClosed || closed.ConsecutiveFailures != 0 || closed.OpenedAt != nil {
		t.Fatalf("unexpected closed state: %+v", closed)
	}
}

func TestChannelHealthHalfOpenFailureReopens(t *testing.T) {
	st, clock := newHealthTestStore()
	for i := 0; i < 5; i++ {
		if _, err := st.RecordChannelFailure(1, domain.FailureUpstream5xx); err != nil {
			t.Fatal(err)
		}
	}
	*clock = clock.Add(30 * time.Second)

	reopened, err := st.RecordChannelFailure(1, domain.FailureUpstream5xx)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.State != domain.HealthOpen {
		t.Fatalf("state = %s, want open", reopened.State)
	}
}

func TestResetChannelHealth(t *testing.T) {
	st, _ := newHealthTestStore()
	if _, err := st.RecordChannelFailure(1, domain.FailureUpstream401); err != nil {
		t.Fatal(err)
	}
	if err := st.ResetChannelHealth(1); err != nil {
		t.Fatal(err)
	}
	health, _ := st.GetChannelHealth(1)
	if health.State != domain.HealthClosed || health.FailureCount != 0 {
		t.Fatalf("after reset: %+v", health)
	}
}

func TestListChannelHealth(t *testing.T) {
	st, _ := newHealthTestStore()
	if _, err := st.CreateChannel(domain.ChannelInput{Name: "channel", BaseURL: "https://channel.test", APIKey: "secret", Status: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordChannelFailure(1, domain.FailureUpstream5xx); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListChannelHealth()
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("total = %d, want 1", list.Total)
	}
	entry := list.List[0]
	if entry.ChannelID != 1 || entry.State != "closed" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestChannelHealthConcurrentFailures(t *testing.T) {
	st, _ := newHealthTestStore()

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.RecordChannelFailure(1, domain.FailureUpstream5xx); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent RecordChannelFailure: %v", err)
	}

	health, err := st.GetChannelHealth(1)
	if err != nil {
		t.Fatal(err)
	}
	if health.ConsecutiveFailures != workers {
		t.Fatalf("consecutive_failures = %d, want %d", health.ConsecutiveFailures, workers)
	}
}

func TestChannelProbeLeaseAllowsOnlyOneConcurrentProbe(t *testing.T) {
	st := NewWithClock(func() time.Time { return time.Unix(100, 0).UTC() })
	first, err := st.AcquireChannelProbe(context.Background(), 1, time.Minute)
	if err != nil || !first {
		t.Fatalf("first probe = %v,%v", first, err)
	}
	second, err := st.AcquireChannelProbe(context.Background(), 1, time.Minute)
	if err != nil || second {
		t.Fatalf("second probe = %v,%v, want denied", second, err)
	}
}

func TestRouteCandidatesExcludeOpenChannel(t *testing.T) {
	st, _ := newHealthTestStore()
	created, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	channelID := created.ID
	if _, err := st.CreateChannelModel(channelID, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	candidates, err := st.RouteCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if candidates.Total != 1 {
		t.Fatalf("candidates total = %d, want 1 before tripping", candidates.Total)
	}

	for i := 0; i < 5; i++ {
		if _, err := st.RecordChannelFailure(channelID, domain.FailureUpstream5xx); err != nil {
			t.Fatal(err)
		}
	}
	candidates, err = st.RouteCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if candidates.Total != 0 {
		t.Fatalf("candidates total = %d, want 0 while open", candidates.Total)
	}
}
