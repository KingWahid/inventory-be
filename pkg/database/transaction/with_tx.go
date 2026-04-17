package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

type TxFunc func(ctx context.Context, tx *sql.Tx) error

func WithTx(ctx context.Context, db *sql.DB, fn TxFunc) (err error) {
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
