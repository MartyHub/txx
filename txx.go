package txx

import (
	"context"
	"database/sql"
)

// Current transaction stored in context.
type Current struct {
	Tx   *sql.Tx
	Opts *sql.TxOptions
}

// IsValid returns if current transaction is valid.
func (c Current) IsValid() bool {
	return c.Tx != nil
}

// NewTransactionRequired returns if a new transaction is required to match given options.
func (c Current) NewTransactionRequired(opts *sql.TxOptions) bool {
	if !c.IsValid() {
		return true
	}

	if c.Opts == nil {
		return opts != nil
	} else if opts == nil {
		return true
	}

	if c.Opts.ReadOnly != opts.ReadOnly {
		return true
	}

	return opts.Isolation > c.Opts.Isolation
}

// ReadOnly returns a read-only transaction option.
func ReadOnly() *sql.TxOptions {
	return &sql.TxOptions{ReadOnly: true}
}

// Ensure function f run in a transaction with given options.
//
// If a transaction already exists matching given options, this transaction is reused,
// otherwise a new transaction is created.
func Ensure(ctx context.Context, db *sql.DB, opts *sql.TxOptions, f func(ctx context.Context) error) error {
	current := Get(ctx)
	if current.NewTransactionRequired(opts) {
		return Wrap(ctx, db, opts, f)
	}

	return f(ctx)
}

// Wrap function f in a new transaction with given options.
//
// If function f returns an error or panic, the transaction is aborted,
// otherwise the transaction is committed.
func Wrap(ctx context.Context, db *sql.DB, opts *sql.TxOptions, f func(ctx context.Context) error) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()

			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = f(Set(ctx, tx, opts))

	return err
}

type key int

var ctxKey key //nolint:gochecknoglobals

// Get the current transaction from given context.
func Get(ctx context.Context) Current {
	if result, ok := ctx.Value(ctxKey).(Current); ok {
		return result
	}

	return Current{}
}

func Set(ctx context.Context, tx *sql.Tx, opts *sql.TxOptions) context.Context {
	return context.WithValue(ctx, ctxKey, Current{
		Tx:   tx,
		Opts: opts,
	})
}
