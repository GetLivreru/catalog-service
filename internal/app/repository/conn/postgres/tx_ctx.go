package rcpostgres

import (
	"context"

	"github.com/uptrace/bun"
)

type contextKeyTx struct{}

func getTxFromContext(ctx context.Context) bun.Tx {
	tx, _ := ctx.Value(contextKeyTx{}).(bun.Tx)
	return tx
}

func setTxToContext(ctx context.Context, tx bun.Tx) context.Context {
	return context.WithValue(ctx, contextKeyTx{}, tx)
}
