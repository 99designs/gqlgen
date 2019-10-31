package graphql

import "context"

const serverCtx key = "serverCtx"

func GetServerContext(ctx context.Context) ExecutableSchema {
	return ctx.Value(serverCtx).(ExecutableSchema)
}

func WithServerContext(ctx context.Context, es ExecutableSchema) context.Context {
	return context.WithValue(ctx, serverCtx, es)
}
