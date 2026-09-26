package catalog

// FailureReason describes why an upstream channel attempt failed. The values
// are shared by the proxy (which classifies upstream results) and the circuit
// breaker (which decides whether to open), so a rename is a compile error
// instead of a silent behaviour change. The zero value means the outcome must
// not be attributed to the channel (e.g. caller errors and client cancellation).
type FailureReason string

const (
	FailureUpstreamUnreachable FailureReason = "upstream_unreachable"
	FailureUpstream401         FailureReason = "upstream_401"
	FailureUpstream402         FailureReason = "upstream_402"
	FailureUpstream403         FailureReason = "upstream_403"
	FailureUpstream429         FailureReason = "upstream_429"
	FailureUpstream5xx         FailureReason = "upstream_5xx"
	FailureUpstreamProtocol    FailureReason = "upstream_protocol_error"
	FailureUpstreamTimeout     FailureReason = "upstream_timeout"
)

// CountsAsChannelFailure reports whether the outcome is attributable to the
// channel. It is the single source of truth for the breaker decision; the zero
// value is the explicit "not a channel failure" sentinel.
func (r FailureReason) CountsAsChannelFailure() bool {
	return r != ""
}

// IsDeterministic reports whether the failure will not recover on retry, so the
// breaker should open immediately instead of counting toward the threshold.
func (r FailureReason) IsDeterministic() bool {
	switch r {
	case FailureUpstream401, FailureUpstream402, FailureUpstream403:
		return true
	default:
		return false
	}
}
