package proxy

import (
	"errors"
	"net/http"
	"testing"

	"LLMGateway/server/internal/domain"
)

func TestClassifyUpstreamResult(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		err     error
		failure bool
		reason  domain.FailureReason
	}{
		{"transport error", 0, errors.New("dial tcp: refused"), true, domain.FailureUpstreamUnreachable},
		{"429", http.StatusTooManyRequests, nil, true, domain.FailureUpstream429},
		{"401", http.StatusUnauthorized, nil, true, domain.FailureUpstream401},
		{"403", http.StatusForbidden, nil, true, domain.FailureUpstream403},
		{"402", http.StatusPaymentRequired, nil, true, domain.FailureUpstream402},
		{"500", http.StatusInternalServerError, nil, true, domain.FailureUpstream5xx},
		{"503", http.StatusServiceUnavailable, nil, true, domain.FailureUpstream5xx},
		{"400", http.StatusBadRequest, nil, false, ""},
		{"404", http.StatusNotFound, nil, false, ""},
		{"422", http.StatusUnprocessableEntity, nil, false, ""},
		{"200", http.StatusOK, nil, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, failure := classifyUpstreamResult(tt.status, tt.err)
			if failure != tt.failure || reason != tt.reason {
				t.Fatalf("classifyUpstreamResult(%d, %v) = (%q, %v), want (%q, %v)", tt.status, tt.err, reason, failure, tt.reason, tt.failure)
			}
		})
	}
}
