package handler

import (
	"net/http"
	"testing"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store/memory"
)

func newRouteTestApp(st *memory.Store, randIntN func(int) int) *Server {
	return &Server{
		store:    st,
		client:   &http.Client{},
		randIntN: randIntN,
		now:      time.Now,
	}
}

func seedRoutingStore(t *testing.T) *memory.Store {
	t.Helper()
	st := memory.New()
	create := func(name string, priority, weight int, balance string) int {
		var balancePtr *string
		if balance != "" {
			balancePtr = &balance
		}
		created, err := st.CreateChannel(domain.ChannelInput{Name: name, BaseURL: "https://" + name + ".test", APIKey: "sk", Status: 1, Priority: priority, Weight: weight, Balance: balancePtr})
		if err != nil {
			t.Fatal(err)
		}
		return created.ID
	}
	low := create("low", 5, 100, "")
	highA := create("highA", 10, 100, "")
	highB := create("highB", 10, 200, "")
	zero := create("zero", 20, 500, "0.000000")
	for _, id := range []int{low, highA, highB, zero} {
		if _, err := st.CreateChannelModel(id, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "up", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	return st
}

func TestSelectChannelUsesHighestPriorityGroup(t *testing.T) {
	st := seedRoutingStore(t)

	// rand 0 selects the first candidate in the highest priority group, which
	// is ordered by weight desc (highB weight 200 before highA weight 100).
	a := newRouteTestApp(st, func(int) int { return 0 })
	candidate, err := a.selectChannel("gpt")
	if err != nil {
		t.Fatalf("selectChannel: %v", err)
	}
	if candidate.ChannelName != "highB" || candidate.Priority != 10 {
		t.Fatalf("candidate = %+v, want highB priority 10", candidate)
	}
	if candidate.Balance != nil {
		t.Fatalf("expected nil balance, got %v", *candidate.Balance)
	}
}

func TestSelectChannelWeightedFallback(t *testing.T) {
	st := seedRoutingStore(t)

	// Total weight in the top group is 300; a pick of 299 lands on highA.
	a := newRouteTestApp(st, func(int) int { return 299 })
	candidate, err := a.selectChannel("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ChannelName != "highA" {
		t.Fatalf("candidate = %+v, want highA", candidate)
	}
}

func TestSelectChannelExcludesNonPositiveBalance(t *testing.T) {
	st := seedRoutingStore(t)

	// Only the zero-balance channel serves "only-zero".
	if _, err := st.CreateChannelModel(4, domain.ChannelModel{ModelName: "only-zero", UpstreamModel: "up", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	a := newRouteTestApp(st, func(int) int { return 0 })
	if _, err := a.selectChannel("only-zero"); err != ErrNoHealthyChannel {
		t.Fatalf("err = %v, want ErrNoHealthyChannel", err)
	}
}

func TestSelectChannelNoCandidates(t *testing.T) {
	a := newRouteTestApp(memory.New(), func(int) int { return 0 })
	if _, err := a.selectChannel("missing"); err != ErrNoHealthyChannel {
		t.Fatalf("err = %v, want ErrNoHealthyChannel", err)
	}
}
