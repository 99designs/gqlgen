package graphql

import (
	"context"
	"fmt"
	"sync"
)

type ErrorPresenterFunc func(context.Context, error) error

func DefaultErrorPresenter(ctx context.Context, err error) error {
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

func (r *ResolverError) Error() string {
	return r.Message
}

type ErrorBuilder struct {
	Errors []error
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
