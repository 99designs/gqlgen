package tracing_logger

import "log"

// Logger is an interface that can be implemented to log errors that occur during the tracing process
// This can use the default Go logger or a custom logger (e.g. logrus or zap)
type Logger interface {
	Print(args ...interface{})
	Println(args ...interface{})
	Printf(format string, args ...interface{})
}

func New() *NoopLogger {
	return &NoopLogger{
		log.New(NullWriter(1), "", log.LstdFlags),
	}
}

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

type NoopLogger struct {
	*log.Logger
}

func (l *NoopLogger) Print(args ...interface{}) {}

func (l *NoopLogger) Printf(format string, args ...interface{}) {}

func (l *NoopLogger) Println(v ...interface{}) {}
