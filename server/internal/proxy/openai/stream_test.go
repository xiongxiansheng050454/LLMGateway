package openai

import (
	"strings"
	"testing"
)

func TestRewriteStreamingRequestForcesUsage(t *testing.T) {
	rewritten, err := rewriteRequest([]byte(`{"model":"public","stream":true,"stream_options":{"include_usage":false,"future":"kept"}}`), "upstream")
	if err != nil {
		t.Fatal(err)
	}
	text := string(rewritten)
	for _, want := range []string{`"model":"upstream"`, `"stream":true`, `"include_usage":true`, `"future":"kept"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("rewritten request = %s, want %s", text, want)
		}
	}
}

func TestParseStreamRewritesModelAndCapturesUsage(t *testing.T) {
	input := strings.Join([]string{
		": keepalive\n\n",
		"data: {\"id\":\"chatcmpl-1\",\"model\":\"upstream\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n",
		"data: {\"id\":\"chatcmpl-1\",\"model\":\"upstream\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12,\"prompt_tokens_details\":{\"cached_tokens\":3}}}\n\n",
		"data: [DONE]\n\n",
	}, "")

	var events []streamEvent
	err := parseStream(strings.NewReader(input), "public", func(event streamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("events = %d, want 4", len(events))
	}
	if events[0].Data || events[0].Done || string(events[0].Frame) != ": keepalive\n\n" {
		t.Fatalf("comment event = %+v", events[0])
	}
	if !events[1].Data || strings.Contains(string(events[1].Frame), "upstream") || !strings.Contains(string(events[1].Frame), `"model":"public"`) {
		t.Fatalf("data event = %s", events[1].Frame)
	}
	if events[2].Usage == nil || events[2].Usage.TotalTokens != 12 || events[2].Usage.CachedInputTokens != 3 {
		t.Fatalf("usage event = %+v", events[2])
	}
	if !events[3].Done || string(events[3].Frame) != "data: [DONE]\n\n" {
		t.Fatalf("done event = %+v", events[3])
	}
}

func TestParseStreamRejectsMalformedJSONData(t *testing.T) {
	err := parseStream(strings.NewReader("data: {not-json}\n\n"), "public", func(streamEvent) error { return nil })
	if err == nil {
		t.Fatal("parseStream accepted malformed JSON data")
	}
}
