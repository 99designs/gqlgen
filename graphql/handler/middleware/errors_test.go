package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/gqlerror"
)

func TestErrorPresenter(t *testing.T) {
	rc := testMiddleware(ErrorPresenter(func(i context.Context, e error) *gqlerror.Error {
		return &gqlerror.Error{Message: "boom"}
	}))

	require.True(t, rc.InvokedNext)
	// cant test for function equality in go, so testing the return type instead
	require.Equal(t, "boom", rc.ResultContext.ErrorPresenter(nil, nil).Message)
}

func TestRecoverFunc(t *testing.T) {
	rc := testMiddleware(RecoverFunc(func(ctx context.Context, err interface{}) (userMessage error) {
		return fmt.Errorf("boom")
	}))

	require.True(t, rc.InvokedNext)
	// cant test for function equality in go, so testing the return type instead
	assert.Equal(t, "boom", rc.ResultContext.Recover(nil, nil).Error())
}
