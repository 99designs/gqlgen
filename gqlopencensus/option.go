package gqlopencensus

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type config struct {
	tracer *tracerImpl
}

// Option is anything that can configure Tracer.
type Option interface {
	apply(cfg *config)
}

type optionFunc func(cfg *config)

func (opt optionFunc) apply(cfg *config) {
	opt(cfg)
}

// WithStartOperationExecution returns option that execute some process on StartOperationExecution step.
func WithStartOperationExecution(f func(ctx context.Context) context.Context) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.startOperationExecutions = append(cfg.tracer.startOperationExecutions, f)
	})
}

// WithStartFieldExecution returns option that execute some process on StartFieldExecution step.
func WithStartFieldExecution(f func(ctx context.Context, field graphql.CollectedField) context.Context) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.startFieldExecutions = append(cfg.tracer.startFieldExecutions, f)
	})
}

// WithStartFieldResolverExecution returns option that execute some process on StartFieldResolverExecution step.
func WithStartFieldResolverExecution(f func(ctx context.Context, rc *graphql.ResolverContext) context.Context) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.startFieldResolverExecutions = append(cfg.tracer.startFieldResolverExecutions, f)
	})
}

// WithStartFieldChildExecution returns option that execute some process on StartFieldChildExecution step.
func WithStartFieldChildExecution(f func(ctx context.Context) context.Context) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.startFieldChildExecutions = append(cfg.tracer.startFieldChildExecutions, f)
	})
}

// WithEndFieldExecution returns option that execute some process on EndFieldExecution step.
func WithEndFieldExecution(f func(ctx context.Context)) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.endFieldExecutions = append(cfg.tracer.endFieldExecutions, f)
	})
}

// WithEndOperationExecutions returns option that execute some process on EndOperationExecutions step.
func WithEndOperationExecutions(f func(ctx context.Context)) Option {
	return optionFunc(func(cfg *config) {
		cfg.tracer.endOperationExecutions = append(cfg.tracer.endOperationExecutions, f)
	})
}
