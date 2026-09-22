package postgres

import (
	"context"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
)

// Tx wraps a PostgreSQL transaction and implements the module transaction
// primitive interfaces. It must stay free of business rules and multi-step
// orchestration; those live in the business modules.
type Tx struct {
	tx      pgx.Tx
	queries *sqlc.Queries
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
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Tx{tx: tx, queries: sqlc.New(tx)}); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// AccountsTx exposes transaction-scoped account primitives.
func (s *Store) AccountsTx() accounts.TxManager { return accountsTxManager{store: s} }
