package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

func TestWithTxSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithTxRollbackOnFnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	expectedErr := errors.New("fn failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected fn error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithTxRollbackOnPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	}()

	_ = WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		panic("boom")
	})
}

func TestWithTxCommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err = WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if !errors.Is(err, errorcodes.ErrTxCommit) {
		t.Fatalf("expected ErrCommitTx, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithTxBeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if !errors.Is(err, errorcodes.ErrTxBegin) {
		t.Fatalf("expected ErrBeginTx, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
