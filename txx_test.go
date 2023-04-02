package txx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func checkTxExists(ctx context.Context) error {
	if !Get(ctx).IsValid() {
		return errors.New("a transaction should exist") //nolint:goerr113
	}

	return nil
}

func checkTxEquals(tx *sql.Tx) func(context.Context) error {
	return func(ctx context.Context) error {
		got := Get(ctx).Tx
		if got != tx {
			return fmt.Errorf("expected %v, got %v", tx, got) //nolint:goerr113
		}

		return nil
	}
}

func fail(_ context.Context) error {
	return errors.New("test") //nolint:goerr113
}

func TestReadOnly(t *testing.T) {
	opts := ReadOnly()

	require.NotNil(t, opts)
	assert.True(t, opts.ReadOnly)
	assert.Equal(t, sql.LevelDefault, opts.Isolation)
}

func TestCurrent_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		current Current
		want    bool
	}{
		{
			name:    "not valid",
			current: Current{},
			want:    false,
		},
		{
			name:    "valid",
			current: Current{Tx: &sql.Tx{}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.current.IsValid())
		})
	}
}

func TestCurrent_NewTransactionRequired(t *testing.T) { //nolint:funlen
	tests := []struct {
		name    string
		current Current
		opts    *sql.TxOptions
		want    bool
	}{
		{
			name:    "nil->nil",
			current: Current{Tx: &sql.Tx{}},
			opts:    nil,
			want:    false,
		},
		{
			name:    "readonly->readonly",
			current: Current{Tx: &sql.Tx{}, Opts: ReadOnly()},
			opts:    ReadOnly(),
			want:    false,
		},
		{
			name:    "readonly->nil",
			current: Current{Tx: &sql.Tx{}, Opts: ReadOnly()},
			opts:    nil,
			want:    true,
		},
		{
			name:    "nil->readonly",
			current: Current{Tx: &sql.Tx{}},
			opts:    ReadOnly(),
			want:    true,
		},
		{
			name:    "readonly->!readonly",
			current: Current{Tx: &sql.Tx{}, Opts: ReadOnly()},
			opts:    &sql.TxOptions{},
			want:    true,
		},
		{
			name:    "!readonly->readonly",
			current: Current{Tx: &sql.Tx{}, Opts: &sql.TxOptions{}},
			opts:    ReadOnly(),
			want:    true,
		},
		{
			name:    "compatible isolation level",
			current: Current{Tx: &sql.Tx{}, Opts: &sql.TxOptions{Isolation: sql.LevelWriteCommitted}},
			opts:    &sql.TxOptions{Isolation: sql.LevelReadCommitted},
			want:    false,
		},
		{
			name:    "isolation level increase",
			current: Current{Tx: &sql.Tx{}, Opts: &sql.TxOptions{Isolation: sql.LevelReadCommitted}},
			opts:    &sql.TxOptions{Isolation: sql.LevelWriteCommitted},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.current.NewTransactionRequired(tt.opts))
		})
	}
}

func TestEnsure(t *testing.T) {
	db := testDB(t)
	tx := &sql.Tx{}

	tests := []struct {
		name    string
		setup   func() context.Context
		f       func(ctx context.Context) error
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "create transaction",
			setup:   context.Background,
			f:       checkTxExists,
			wantErr: assert.NoError,
		},
		{
			name:    "error",
			setup:   context.Background,
			f:       fail,
			wantErr: assert.Error,
		},
		{
			name: "use existing transaction",
			setup: func() context.Context {
				return Set(context.Background(), tx, nil)
			},
			f:       checkTxEquals(tx),
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Ensure(tt.setup(), db, nil, tt.f)

			tt.wantErr(t, err)
		})
	}
}

func TestWrap(t *testing.T) {
	db := testDB(t)
	tests := []struct {
		name    string
		f       func(ctx context.Context) error
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "no error",
			f:       checkTxExists,
			wantErr: assert.NoError,
		},
		{
			name:    "error",
			f:       fail,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(context.Background(), db, nil, tt.f)

			tt.wantErr(t, err)
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name  string
		setup func() context.Context
		valid bool
	}{
		{
			name:  "empty",
			setup: context.Background,
			valid: false,
		},
		{
			name: "exist",
			setup: func() context.Context {
				return Set(context.Background(), &sql.Tx{}, nil)
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, Get(tt.setup()).IsValid())
		})
	}
}

func TestSet(t *testing.T) {
	ctx := context.Background()
	tx := &sql.Tx{}
	opts := ReadOnly()

	txCtx := Set(ctx, tx, opts)

	assert.Nil(t, ctx.Value(ctxKey))

	require.NotNil(t, txCtx)
	assert.NotEqual(t, ctx, txCtx)

	current := Get(txCtx)

	require.NotNil(t, current)
	assert.Equal(t, tx, current.Tx)
	assert.Equal(t, opts, current.Opts)
}
