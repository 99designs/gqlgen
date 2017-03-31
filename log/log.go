package log

import (
	"context"
	"log"
	"runtime"
)

// Logger is the interface used to log panics that occur durring query execution. It is setable via graphql.ParseSchema
type Logger interface {
	LogPanic(ctx context.Context, value interface{})
}

// DefaultLogger is the default logger used to log panics that occur durring query execution
type DefaultLogger struct{}

// LogPanic is used to log recovered panic values that occur durring query execution
func (l *DefaultLogger) LogPanic(_ context.Context, value interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("graphql: panic occurred: %v\n%s", value, buf)
}
