package logger_test

import (
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/logger"
	"github.com/stretchr/testify/assert"
)

func TestNoopLogger_Print(t *testing.T) {
	l := logger.NewNoopLogger()
	assert.NotPanics(t, func() {
		l.Print("test")
	})
}

func TestNoopLogger_Printf(t *testing.T) {
	l := logger.NewNoopLogger()
	assert.NotPanics(t, func() {
		l.Printf("test %s", "formatted")
	})
}

func TestNoopLogger_Println(t *testing.T) {
	l := logger.NewNoopLogger()
	assert.NotPanics(t, func() {
		l.Println("test")
	})
}
