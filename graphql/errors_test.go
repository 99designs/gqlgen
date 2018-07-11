package graphql

import (
	"context"
	"strings"
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

	errs := sliceErr{&testErr{"err3"}, &testErr{"err4"}}
	b.Error(context.Background(), errs)

	require.Len(t, b.Errors, 4)
	assert.EqualError(t, b.Errors[0], "err1")
	assert.EqualError(t, b.Errors[1], "err2 public")
	assert.EqualError(t, b.Errors[2], "err3")
	assert.EqualError(t, b.Errors[3], "err4")
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

type sliceErr []*testErr

func (err sliceErr) Error() string {
	var errs []string
	for _, err := range err {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, ";")
}

func (err sliceErr) Errors() (errs []error) {
	for _, err := range err {
		errs = append(errs, err)
	}
	return errs
}

func convertErr(ctx context.Context, err error) error {
	if errConv, ok := err.(*publicErr); ok {
		return &ResolverError{Message: errConv.public}
	}
	return &ResolverError{Message: err.Error()}
}
