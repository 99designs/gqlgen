package graphql

import (
	"context"
	"fmt"
	"sync"
)

// MarshalableError represents an error that can be encoded by the transport layer and sent to the client. In this
// package everything should be transport-agnostic, so we cant make any assertions on what is expected.
//
// For the packaged handler implementation the errors must be json.Marshall'able, for a custom protobuf based transport
// the returned type should be protobufable
type MarshalableError interface{}

type ErrorPresenterFunc func(context.Context, error) MarshalableError

func DefaultErrorPresenter(ctx context.Context, err error) MarshalableError {
	return &ResolverError{
		Message: err.Error(),
		Path:    GetResolverContext(ctx).Path,
	}
}

// ResolverError is the default error type returned by ErrorPresenter. You can replace it with your own by returning
// something different from the ErrorPresenter
type ResolverError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

type ErrorBuilder struct {
	Errors []MarshalableError
	// ErrorPresenter will be used to generate the error
	// message from errors given to Error().
	ErrorPresenter ErrorPresenterFunc
	mu             sync.Mutex
}

func (c *ErrorBuilder) Errorf(ctx context.Context, format string, args ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, fmt.Errorf(format, args...)))
}

func (c *ErrorBuilder) Error(ctx context.Context, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, err))
}
