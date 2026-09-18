package domain

// FailureReason describes why an upstream channel attempt failed. The values
// are shared by the proxy (which classifies upstream results) and the circuit
// breaker (which decides whether to open), so a rename is a compile error
// instead of a silent behaviour change.
type FailureReason string

const (
	FailureUpstreamUnreachable FailureReason = "upstream_unreachable"
	FailureUpstream401         FailureReason = "upstream_401"
	FailureUpstream402         FailureReason = "upstream_402"
	FailureUpstream403         FailureReason = "upstream_403"
	FailureUpstream429         FailureReason = "upstream_429"
	FailureUpstream5xx         FailureReason = "upstream_5xx"
	FailureUpstreamProtocol    FailureReason = "upstream_protocol_error"
	FailureInvalidChannelKey   FailureReason = "invalid_channel_key"
	FailureUpstreamTimeout     FailureReason = "upstream_timeout"
	FailureCaller400           FailureReason = "caller_400"
	FailureCaller404           FailureReason = "caller_404"
	FailureCaller422           FailureReason = "caller_422"
	FailureClientCanceled      FailureReason = "client_canceled"
)

// IsDeterministic reports whether the failure will not recover on retry, so the
// breaker should open immediately instead of counting toward the threshold.
func (r FailureReason) IsDeterministic() bool {
	switch r {
	case FailureUpstream401, FailureUpstream402, FailureUpstream403, FailureInvalidChannelKey:
		return true
	default:
		return false
	}
}

func (r FailureReason) CountsAsChannelFailure() bool {
	switch r {
	case FailureCaller400, FailureCaller404, FailureCaller422, FailureClientCanceled:
		return false
	default:
		return true
	}
}
