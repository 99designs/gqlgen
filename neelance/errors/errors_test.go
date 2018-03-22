package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Error(t *testing.T) {
	t.Run("with err converter", func(t *testing.T) {
		b := Builder{ErrorMessageFn: convertErr}
		b.Error(&testErr{"err1"})
		b.Error(&publicErr{
			message: "err2",
			public:  "err2 public",
		})

		require.Len(t, b.Errors, 2)
		assert.EqualError(t, b.Errors[0], "graphql: err1")
		assert.EqualError(t, b.Errors[1], "graphql: err2 public")
	})
	t.Run("without err converter", func(t *testing.T) {
		var b Builder
		b.Error(&testErr{"err1"})
		b.Error(&publicErr{
			message: "err2",
			public:  "err2 public",
		})

		require.Len(t, b.Errors, 2)
		assert.EqualError(t, b.Errors[0], "graphql: err1")
		assert.EqualError(t, b.Errors[1], "graphql: err2")
	})
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

func convertErr(err error) string {
	if errConv, ok := err.(*publicErr); ok {
		return errConv.public
	}
	return err.Error()
}
