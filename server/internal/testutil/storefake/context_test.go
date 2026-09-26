package storefake

import (
	"context"
	"testing"

	"LLMGateway/server/internal/accounts"
	domain "LLMGateway/server/internal/testutil/testtypes"
)

// recordingTxManager captures the context a business service passes to InTx so
// tests can assert that request context is propagated rather than replaced with
// context.Background.
type recordingTxManager struct {
	inner accounts.TxManager
	got   context.Context
}

func (r *recordingTxManager) InTx(ctx context.Context, fn func(accounts.Tx) error) error {
	r.got = ctx
	return r.inner.InTx(ctx, fn)
}

func TestBusinessServicePropagatesContextToTx(t *testing.T) {
	st := New()
	tm := &recordingTxManager{inner: st.AccountsTx()}
	acc := accounts.New(st, tm)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	if _, err := acc.CreateUser(ctx, domain.UserInput{Nickname: "Alice"}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if tm.got == nil {
		t.Fatal("TxManager did not receive a context")
	}
	if tm.got.Value(ctxKey{}) != "sentinel" {
		t.Fatalf("context value not propagated through business service")
	}
}
