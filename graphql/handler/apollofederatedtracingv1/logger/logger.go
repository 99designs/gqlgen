package logger

import "log"

// Logger is an interface that can be implemented to log errors that occur during the tracing process
// This can use the default Go logger or a custom logger (e.g. logrus or zap)
type Logger interface {
	Print(args any)
	Println(args any)
	Printf(format string, args any)
}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{
		log.New(NullWriter(1), "", log.LstdFlags),
	}
}

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

type NoopLogger struct {
	*log.Logger
}

func (l *NoopLogger) Print(args any) {}

func (l *NoopLogger) Printf(format string, args any) {}

func (l *NoopLogger) Println(v any) {}
