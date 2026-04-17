package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWithSQLTxSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = WithSQLTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithSQLTx returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithSQLTxRollbackOnFnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	expectedErr := errors.New("fn failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = WithSQLTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected fn error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithSQLTxRollbackOnPanic(t *testing.T) {
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

	_ = WithSQLTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		panic("boom")
	})
}

func TestWithSQLTxCommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err = WithSQLTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if !errors.Is(err, errorcodes.ErrTxCommit) {
		t.Fatalf("expected ErrCommitTx, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithSQLTxBeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = WithSQLTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		return nil
	})
	if !errors.Is(err, errorcodes.ErrTxBegin) {
		t.Fatalf("expected ErrBeginTx, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestGetDBFallback(t *testing.T) {
	db, _, err := gormMockDB(t)
	if err != nil {
		t.Fatalf("gormMockDB: %v", err)
	}

	got := GetDB(context.Background(), db)
	if got != db {
		t.Fatalf("expected default db fallback")
	}
}

func TestWithTxAndGetDB(t *testing.T) {
	db, _, err := gormMockDB(t)
	if err != nil {
		t.Fatalf("gormMockDB: %v", err)
	}

	ctx := WithTx(context.Background(), db)
	got := GetDB(ctx, nil)
	if got != db {
		t.Fatalf("expected tx from context")
	}
}

func TestRunInTxSuccess(t *testing.T) {
	db, mock, err := gormMockDB(t)
	if err != nil {
		t.Fatalf("gormMockDB: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = RunInTx(context.Background(), db, func(ctx context.Context) error {
		if GetDB(ctx, nil) == nil {
			t.Fatal("expected tx in context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunInTx returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestRunInTxRollbackOnError(t *testing.T) {
	db, mock, err := gormMockDB(t)
	if err != nil {
		t.Fatalf("gormMockDB: %v", err)
	}
	expectedErr := errors.New("fn failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = RunInTx(context.Background(), db, func(ctx context.Context) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected fn error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestRunInTxNestedReuse(t *testing.T) {
	db, mock, err := gormMockDB(t)
	if err != nil {
		t.Fatalf("gormMockDB: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = RunInTx(context.Background(), db, func(outerCtx context.Context) error {
		outerTx := GetDB(outerCtx, nil)
		if outerTx == nil {
			t.Fatal("expected outer tx")
		}
		return RunInTx(outerCtx, db, func(innerCtx context.Context) error {
			innerTx := GetDB(innerCtx, nil)
			if innerTx == nil {
				t.Fatal("expected inner tx")
			}
			if innerTx != outerTx {
				t.Fatal("expected nested RunInTx to reuse same tx")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("RunInTx nested returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func gormMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, error) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	return gdb, mock, nil
}
