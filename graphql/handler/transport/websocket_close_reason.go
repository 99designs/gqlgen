package transport

import (
	"context"

	"github.com/gorilla/websocket"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var closeReasonCtxKey = &wsCloseReasonContextKey{"close-reason"}

type wsCloseReasonContextKey struct {
	name string
}

func AppendCloseReason(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, closeReasonCtxKey, reason)
}

func closeReasonForContext(ctx context.Context) string {
	reason, _ := ctx.Value(closeReasonCtxKey).(string)
	return reason
}

type contextKeyWebsocketCloseCode struct{}

func WithWebsocketCloseCode(ctx context.Context, v int) context.Context {
	return context.WithValue(ctx, contextKeyWebsocketCloseCode{}, v)
}

func websocketCloseCodeForContext(ctx context.Context) int {
	var code, ok = ctx.Value(contextKeyWebsocketCloseCode{}).(int)
	if !ok {
		return websocket.CloseNormalClosure
	}
	return code
}
