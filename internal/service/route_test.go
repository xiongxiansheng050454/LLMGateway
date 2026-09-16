package service

import (
	"net/http"
	"testing"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/store/memory"
)

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
		return created["id"].(int)
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
	proxy := New(st, &http.Client{}, WithRandSource(func(int) int { return 0 }))
	candidate, err := proxy.SelectChannel("gpt")
	if err != nil {
		t.Fatalf("SelectChannel: %v", err)
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
	proxy := New(st, &http.Client{}, WithRandSource(func(int) int { return 299 }))
	candidate, err := proxy.SelectChannel("gpt")
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
	created, err := st.CreateChannelModel(4, domain.ChannelModel{ModelName: "only-zero", UpstreamModel: "up", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = created

	proxy := New(st, &http.Client{}, WithRandSource(func(int) int { return 0 }))
	if _, err := proxy.SelectChannel("only-zero"); err != ErrNoChannel {
		t.Fatalf("err = %v, want ErrNoChannel", err)
	}
}

func TestSelectChannelNoCandidates(t *testing.T) {
	st := memory.New()
	proxy := New(st, &http.Client{})
	if _, err := proxy.SelectChannel("missing"); err != ErrNoChannel {
		t.Fatalf("err = %v, want ErrNoChannel", err)
	}
}
