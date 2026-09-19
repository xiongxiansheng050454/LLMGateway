package proxy

import (
	"net/http"
	"testing"
	"time"

	"LLMGateway/server/internal/testutil/storefake"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func newRouteTestApp(st *storefake.Store, randIntN func(int) int) *Service {
	return &Service{
		store:    st,
		client:   &http.Client{},
		randIntN: randIntN,
		now:      time.Now,
	}
}

func seedRoutingStore(t *testing.T) *storefake.Store {
	t.Helper()
	st := storefake.New()
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

func TestSelectChannelExcludesBalanceBelowConfiguredReserve(t *testing.T) {
	st := storefake.New()
	lowBalance := "0.999999"
	highBalance := "1.000000"
	for _, input := range []domain.ChannelInput{
		{Name: "below-reserve", BaseURL: "https://below.test", APIKey: "sk", Status: 1, Priority: 10, Weight: 100, Balance: &lowBalance},
		{Name: "at-reserve", BaseURL: "https://at.test", APIKey: "sk", Status: 1, Priority: 10, Weight: 100, Balance: &highBalance},
		{Name: "unlimited", BaseURL: "https://unlimited.test", APIKey: "sk", Status: 1, Priority: 10, Weight: 100},
	} {
		channel, err := st.CreateChannel(input)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateChannelModel(channel.ID, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "gpt", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}

	a := newRouteTestApp(st, func(int) int { return 0 })
	a.ConfigureMinimumRouteBalance("1.000000")
	candidates, err := a.orderedCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].ChannelName == "below-reserve" || candidates[1].ChannelName == "below-reserve" {
		t.Fatalf("candidates = %+v, want channels at-reserve and unlimited", candidates)
	}
}

func TestSelectChannelNoCandidates(t *testing.T) {
	a := newRouteTestApp(storefake.New(), func(int) int { return 0 })
	if _, err := a.selectChannel("missing"); err != ErrNoHealthyChannel {
		t.Fatalf("err = %v, want ErrNoHealthyChannel", err)
	}
}

func TestOrderedCandidatesDeduplicateChannelsAndKeepPriorityFallbacks(t *testing.T) {
	st := storefake.New()
	ids := []int{}
	for _, input := range []domain.ChannelInput{
		{Name: "preferred", BaseURL: "http://preferred", APIKey: "x", Status: 1, Priority: 10, Weight: 10},
		{Name: "fallback", BaseURL: "http://fallback", APIKey: "x", Status: 1, Priority: 5, Weight: 1},
	} {
		created, err := st.CreateChannel(input)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, created.ID)
	}
	for _, channelID := range []int{ids[0], ids[1]} {
		if _, err := st.CreateChannelModel(channelID, domain.ChannelModel{ModelName: "gpt", UpstreamModel: "gpt", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(st, nil, func(int) int { return 0 }, time.Now)
	candidates, err := service.orderedCandidates("gpt")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].ChannelID != ids[0] || candidates[1].ChannelID != ids[1] {
		t.Fatalf("candidates = %+v, want channels 1,2 once", candidates)
	}
}

func TestOrderedCandidatesSticksAPIKeyAndModelToSameChannel(t *testing.T) {
	st := seedRoutingStore(t)
	a := newRouteTestApp(st, func(int) int { return 299 })
	first, err := a.orderedCandidates("gpt", 42)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.orderedCandidates("gpt", 42)
	if err != nil {
		t.Fatal(err)
	}
	if first[0].ChannelID != second[0].ChannelID {
		t.Fatalf("sticky first candidates = %d,%d, want same channel", first[0].ChannelID, second[0].ChannelID)
	}
	otherKey, err := a.orderedCandidates("gpt", 43)
	if err != nil {
		t.Fatal(err)
	}
	if otherKey[0].ChannelID == 0 {
		t.Fatalf("other key candidate = %+v", otherKey)
	}
}
