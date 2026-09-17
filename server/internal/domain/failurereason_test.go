package domain

import "testing"

func TestFailureReasonIsDeterministic(t *testing.T) {
	tests := []struct {
		reason        FailureReason
		deterministic bool
	}{
		{FailureUpstream401, true},
		{FailureUpstream402, true},
		{FailureUpstream403, true},
		{FailureInvalidChannelKey, true},
		{FailureUpstream429, false},
		{FailureUpstream5xx, false},
		{FailureUpstreamUnreachable, false},
		{FailureReason("something_else"), false},
	}
	for _, tt := range tests {
		if got := tt.reason.IsDeterministic(); got != tt.deterministic {
			t.Fatalf("%q.IsDeterministic() = %v, want %v", tt.reason, got, tt.deterministic)
		}
	}
}
