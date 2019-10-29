package graphql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/vektah/gqlparser/gqlerror"
)

type RecoverFunc func(ctx context.Context, err interface{}) (userMessage error)

func DefaultRecover(ctx context.Context, err interface{}) error {
	fmt.Fprintln(os.Stderr, err)
	fmt.Fprintln(os.Stderr)
	debug.PrintStack()

	return errors.New("internal system error")
}

var _ RequestContextMutator = RecoverFunc(nil)

func (f RecoverFunc) MutateRequestContext(ctx context.Context, rc *RequestContext) *gqlerror.Error {
	rc.Recover = f
	return nil
}
