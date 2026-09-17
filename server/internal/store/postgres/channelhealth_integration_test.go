package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
	"LLMGateway/server/internal/store/memory"
)

func createHealthTestChannel(t *testing.T, st interface {
	CreateChannel(domain.ChannelInput) (domain.ChannelDTO, error)
	CreateChannelModel(int, domain.ChannelModel) (domain.ChannelModel, error)
}) int {
	t.Helper()
	created, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	channelID := created.ID
	if _, err := st.CreateChannelModel(channelID, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	return channelID
}

func TestPGChannelHealthLifecycle(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	channelID := createHealthTestChannel(t, st)

	health, err := st.GetChannelHealth(channelID)
	if err != nil {
		t.Fatalf("GetChannelHealth: %v", err)
	}
	if health.State != domain.HealthClosed {
		t.Fatalf("initial state = %s, want closed", health.State)
	}

	for i := 0; i < 5; i++ {
		health, err = st.RecordChannelFailure(channelID, domain.FailureUpstream5xx)
		if err != nil {
			t.Fatalf("RecordChannelFailure: %v", err)
		}
	}
	if health.State != domain.HealthOpen || health.OpenedAt == nil {
		t.Fatalf("state = %s, want open with opened_at", health.State)
	}

	candidates, err := st.RouteCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if candidates.Total != 0 {
		t.Fatalf("candidates total = %d, want 0 while open", candidates.Total)
	}

	// Backdate opened_at so the cooldown has elapsed; the channel is allowed again.
	if _, err := st.pool.Exec(ctx, "UPDATE channel_health SET opened_at = now() - interval '1 minute' WHERE channel_id = $1", channelID); err != nil {
		t.Fatal(err)
	}
	candidates, err = st.RouteCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if candidates.Total != 1 {
		t.Fatalf("candidates total = %d, want 1 after cooldown", candidates.Total)
	}

	closed, err := st.RecordChannelSuccess(channelID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.State != domain.HealthClosed || closed.ConsecutiveFailures != 0 || closed.OpenedAt != nil {
		t.Fatalf("unexpected closed state: %+v", closed)
	}

	if err := st.ResetChannelHealth(channelID); err != nil {
		t.Fatalf("ResetChannelHealth: %v", err)
	}
	reset, _ := st.GetChannelHealth(channelID)
	if reset.State != domain.HealthClosed || reset.FailureCount != 0 {
		t.Fatalf("after reset: %+v", reset)
	}
}

func TestPGChannelHealthHalfOpen(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	channelID := createHealthTestChannel(t, st)

	for i := 0; i < 5; i++ {
		if _, err := st.RecordChannelFailure(channelID, domain.FailureUpstream5xx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.pool.Exec(ctx, "UPDATE channel_health SET opened_at = now() - interval '1 minute' WHERE channel_id = $1", channelID); err != nil {
		t.Fatal(err)
	}

	health, err := st.GetChannelHealth(channelID)
	if err != nil {
		t.Fatal(err)
	}
	if health.State != domain.HealthHalfOpen {
		t.Fatalf("state = %s, want half-open after cooldown", health.State)
	}

	reopened, err := st.RecordChannelFailure(channelID, domain.FailureUpstream5xx)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.State != domain.HealthOpen {
		t.Fatalf("state = %s, want open after half-open failure", reopened.State)
	}
}

func TestPGChannelHealthConcurrentFailures(t *testing.T) {
	st := testStore(t)
	channelID := createHealthTestChannel(t, st)

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.RecordChannelFailure(channelID, domain.FailureUpstream5xx); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent RecordChannelFailure: %v", err)
	}

	health, err := st.GetChannelHealth(channelID)
	if err != nil {
		t.Fatal(err)
	}
	if health.ConsecutiveFailures != workers {
		t.Fatalf("consecutive_failures = %d, want %d (lost updates)", health.ConsecutiveFailures, workers)
	}
	if health.FailureCount != int64(workers) {
		t.Fatalf("failure_count = %d, want %d", health.FailureCount, workers)
	}
	if health.State != domain.HealthOpen {
		t.Fatalf("state = %s, want open", health.State)
	}
}

type healthSnapshot struct {
	state             string
	consecutive       int
	successCount      int64
	failureCount      int64
	openedAtSet       bool
	stateAfterCool    string
	deterministicOpen string
}

func runHealthScenario(t *testing.T, st store.Store, clock *time.Time) healthSnapshot {
	t.Helper()

	created, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	channelID := created.ID

	var health domain.ChannelHealth
	for i := 0; i < 5; i++ {
		health, err = st.RecordChannelFailure(channelID, domain.FailureUpstream5xx)
		if err != nil {
			t.Fatal(err)
		}
	}

	snapshot := healthSnapshot{
		state:        string(health.State),
		consecutive:  health.ConsecutiveFailures,
		successCount: health.SuccessCount,
		failureCount: health.FailureCount,
		openedAtSet:  health.OpenedAt != nil,
	}

	*clock = clock.Add(30 * time.Second)
	afterCool, err := st.GetChannelHealth(channelID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.stateAfterCool = string(afterCool.State)

	// Deterministic failure on a fresh channel opens immediately.
	second, err := st.CreateChannel(domain.ChannelInput{Name: "Other", BaseURL: "https://other.test", APIKey: "sk", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	det, err := st.RecordChannelFailure(second.ID, domain.FailureUpstream401)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.deterministicOpen = string(det.State)

	return snapshot
}

func TestMemoryVsPGChannelHealthConsistency(t *testing.T) {
	fixed := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	memClock := fixed
	mem := memory.NewWithClock(func() time.Time { return memClock })
	memSnapshot := runHealthScenario(t, mem, &memClock)

	pg := testStore(t)
	pgClock := fixed
	pg.now = func() time.Time { return pgClock }
	pgSnapshot := runHealthScenario(t, pg, &pgClock)

	if memSnapshot != pgSnapshot {
		t.Fatalf("memory and PG differ:\nmem = %+v\npg  = %+v", memSnapshot, pgSnapshot)
	}
}
