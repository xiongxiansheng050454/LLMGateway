package proxy

import (
	"errors"
	"net/http"
	"testing"

	domain "LLMGateway/server/internal/testutil/testtypes"
)

func TestClassifyUpstreamResult(t *testing.T) {
	tests := []struct {
		name   string
		status int
		err    error
		reason domain.FailureReason
	}{
		{"transport error", 0, errors.New("dial tcp: refused"), domain.FailureUpstreamUnreachable},
		{"429", http.StatusTooManyRequests, nil, domain.FailureUpstream429},
		{"401", http.StatusUnauthorized, nil, domain.FailureUpstream401},
		{"403", http.StatusForbidden, nil, domain.FailureUpstream403},
		{"402", http.StatusPaymentRequired, nil, domain.FailureUpstream402},
		{"500", http.StatusInternalServerError, nil, domain.FailureUpstream5xx},
		{"503", http.StatusServiceUnavailable, nil, domain.FailureUpstream5xx},
		{"400", http.StatusBadRequest, nil, ""},
		{"404", http.StatusNotFound, nil, ""},
		{"422", http.StatusUnprocessableEntity, nil, ""},
		{"200", http.StatusOK, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := classifyUpstreamResult(tt.status, tt.err)
			if reason != tt.reason {
				t.Fatalf("classifyUpstreamResult(%d, %v) = %q, want %q", tt.status, tt.err, reason, tt.reason)
			}
			if counts := reason.CountsAsChannelFailure(); counts != (tt.reason != "") {
				t.Fatalf("CountsAsChannelFailure(%q) = %v, want %v", reason, counts, tt.reason != "")
			}
		})
	}
}
