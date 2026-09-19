package catalog_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/testutil/storefake"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

func TestChannelTestTimeoutReturnsSafeError(t *testing.T) {
	st := storefake.New()
	channel, err := st.CreateChannel(domain.ChannelInput{Name: "slow", BaseURL: "https://upstream.test", APIKey: "sk-secret", AuthType: "bearer", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateChannelModel(channel.ID, domain.ChannelModel{ModelName: "public", UpstreamModel: "upstream", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	server := catalog.New(st, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})})
	server.ConfigureTestTimeout(10 * time.Millisecond)
	req := httptest.NewRequest(http.MethodPost, "/admin/channels/1/test", bytes.NewBufferString(`{}`))
	result, _, _, _ := server.TestChannel(req, channel.ID)
	item := result.(domain.ChannelTestResultDTO).List[0]
	if item.OK || item.HTTPStatus != 0 || item.Error != "upstream request timed out" || item.LatencyMs <= 0 {
		t.Fatalf("timeout item = %+v", item)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChannelTestCheckAllFalseOnlyTestsFirstEnabledModel(t *testing.T) {
	var models []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		models = append(models, body.Model)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	st := storefake.New()
	channel, err := st.CreateChannel(domain.ChannelInput{Name: "test", BaseURL: upstream.URL, APIKey: "sk", AuthType: "bearer", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []domain.ChannelModel{{ModelName: "first", UpstreamModel: "first-upstream", Enabled: true}, {ModelName: "second", UpstreamModel: "second-upstream", Enabled: true}} {
		if _, err := st.CreateChannelModel(channel.ID, model); err != nil {
			t.Fatal(err)
		}
	}
	server := catalog.New(st, &http.Client{})
	checkAll := false
	body, _ := json.Marshal(map[string]bool{"check_all": checkAll})
	req := httptest.NewRequest(http.MethodPost, "/admin/channels/1/test", bytes.NewReader(body))
	result, _, _, _ := server.TestChannel(req, channel.ID)
	items := result.(domain.ChannelTestResultDTO).List
	if len(items) != 1 || len(models) != 1 || models[0] != "first-upstream" {
		t.Fatalf("items/models = %+v/%+v", items, models)
	}
}
