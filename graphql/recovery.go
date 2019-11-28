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

var _ OperationContextMutator = RecoverFunc(nil)

func (f RecoverFunc) ExtensionName() string {
	return "RecoverFunc"
}

func (f RecoverFunc) Validate(schema ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("RecoverFunc can not be nil")
	}
	return nil
}

func (f RecoverFunc) MutateOperationContext(ctx context.Context, rc *OperationContext) *gqlerror.Error {
	rc.Recover = f
	return nil
}
