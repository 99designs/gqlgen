package graphql

import "context"

type SubscriptionField[T any] interface {
	GetContext() context.Context
	GetField() T
}

func NewSubscriptionField[T any](ctx context.Context, field T) SubscriptionField[T] {
	return &subscriptionFieldImpl[T]{ctx, field}
}

type subscriptionFieldImpl[T any] struct {
	ctx   context.Context
	field T
}

func (s *subscriptionFieldImpl[T]) GetContext() context.Context {
	return s.ctx
}

func (s *subscriptionFieldImpl[T]) GetField() T {
	return s.field
}
