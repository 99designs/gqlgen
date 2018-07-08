package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Error(t *testing.T) {
	b := ErrorBuilder{ErrorPresenter: convertErr}
	b.Error(context.Background(), &testErr{"err1"})
	b.Error(context.Background(), &publicErr{
		message: "err2",
		public:  "err2 public",
	})

	require.Len(t, b.Errors, 2)
	assert.Equal(t, b.Errors[0], &ResolverError{Message: "err1"})
	assert.Equal(t, b.Errors[1], &ResolverError{Message: "err2 public"})
}

type testErr struct {
	message string
}

func (err *testErr) Error() string {
	return err.message
}

type publicErr struct {
	message string
	public  string
}

func (err *publicErr) Error() string {
	return err.message
}

func (err *publicErr) PublicError() string {
	return err.public
}

func convertErr(ctx context.Context, err error) MarshalableError {
	if errConv, ok := err.(*publicErr); ok {
		return &ResolverError{Message: errConv.public}
	}
	return &ResolverError{Message: err.Error()}
}
