package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
	"gorm.io/gorm"
)

type contextKey string

const txContextKey contextKey = "tx:gorm"

// SQLTxFunc is a legacy callback signature for sql.Tx-based flows.
type SQLTxFunc func(ctx context.Context, tx *sql.Tx) error

// WithTx stores a GORM transaction in context.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

// GetDB returns transaction from context if present, otherwise defaultDB.
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if ctx == nil {
		return defaultDB
	}
	tx, ok := ctx.Value(txContextKey).(*gorm.DB)
	if !ok || tx == nil {
		return defaultDB
	}
	return tx
}

// RunInTx executes fn in a transaction. If ctx already has a tx, it reuses it.
func RunInTx(ctx context.Context, db *gorm.DB, fn func(context.Context) error) error {
	if db == nil {
		return errors.New("transaction: gorm db is nil")
	}
	if existing := GetDB(ctx, nil); existing != nil {
		return fn(ctx)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}

// WithSQLTx executes fn with a sql.Tx.
// Deprecated: Migrate callers to RunInTx/WithTx/GetDB using *gorm.DB and context propagation.
func WithSQLTx(ctx context.Context, db *sql.DB, fn SQLTxFunc) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", errorcodes.ErrTxBegin, err)
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				panic(fmt.Errorf("%w: %v (panic: %v)", errorcodes.ErrTxRollback, rbErr, p))
			}
			panic(p)
		}
	}()

	if err = fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return fmt.Errorf("%w: %v", errorcodes.ErrTxRollback, rbErr)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", errorcodes.ErrTxCommit, err)
	}

	return nil
}
