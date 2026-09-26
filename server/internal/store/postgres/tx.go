package postgres

import (
	"context"
	"time"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/db/sqlc"
	settlement "LLMGateway/server/internal/proxy/settlement"
	"LLMGateway/server/internal/quota"

	"github.com/jackc/pgx/v5"
)

// Tx wraps a PostgreSQL transaction and implements the module transaction
// primitive interfaces. It must stay free of business rules and multi-step
// orchestration; those live in the business modules.
type Tx struct {
	ctx     context.Context
	tx      pgx.Tx
	queries *sqlc.Queries
	now     func() time.Time
}

// accountsTxManager adapts Store to accounts.TxManager. A distinct type per
// module is required because Go does not allow one type to implement several
// InTx methods that differ only in the callback signature.
type accountsTxManager struct {
	store *Store
}

func (m accountsTxManager) InTx(ctx context.Context, fn func(accounts.Tx) error) error {
	tx, err := m.store.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(m.store.newTx(ctx, tx)); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// AccountsTx exposes transaction-scoped account primitives.
func (s *Store) AccountsTx() accounts.TxManager { return accountsTxManager{store: s} }

// settlementTxManager adapts Store to settlement.TxManager.
type settlementTxManager struct {
	store *Store
}

func (m settlementTxManager) InTx(ctx context.Context, fn func(settlement.Tx) error) error {
	tx, err := m.store.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(m.store.newTx(ctx, tx)); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// SettlementTx exposes transaction-scoped settlement primitives.
func (s *Store) SettlementTx() settlement.TxManager { return settlementTxManager{store: s} }

// catalogTxManager adapts Store to catalog.TxManager.
type catalogTxManager struct {
	store *Store
}

func (m catalogTxManager) InTx(ctx context.Context, fn func(catalog.Tx) error) error {
	tx, err := m.store.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(m.store.newTx(ctx, tx)); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// CatalogTx exposes transaction-scoped catalog primitives.
func (s *Store) CatalogTx() catalog.TxManager { return catalogTxManager{store: s} }

// quotaTxManager adapts Store to quota.TxManager.
type quotaTxManager struct {
	store *Store
}

func (m quotaTxManager) InTx(ctx context.Context, fn func(quota.Tx) error) error {
	tx, err := m.store.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(m.store.newTx(ctx, tx)); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// QuotaTx exposes transaction-scoped quota primitives.
func (s *Store) QuotaTx() quota.TxManager { return quotaTxManager{store: s} }

func (s *Store) newTx(ctx context.Context, tx pgx.Tx) *Tx {
	return &Tx{ctx: ctx, tx: tx, queries: sqlc.New(tx), now: s.now}
}
