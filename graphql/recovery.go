package graphql

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
)

type RecoverFunc func(err interface{}) (userMessage error)

func DefaultRecoverFunc(err interface{}) error {
	fmt.Fprintln(os.Stderr, err)
	fmt.Fprintln(os.Stderr)
	debug.PrintStack()

	return errors.New("internal system error")
}
