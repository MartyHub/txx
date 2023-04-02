# Txx

`Txx` is a small lib to manage SQL transaction in Go context.  

![build](https://github.com/MartyHub/txx/actions/workflows/go.yml/badge.svg)

## Usage

```go
txx.Ensure(ctx, db, txx.ReadOnly(), func (ctx context.Context) {
	// In a read-only transaction 
	tx := txx.Get(ctx).Tx
})
```
