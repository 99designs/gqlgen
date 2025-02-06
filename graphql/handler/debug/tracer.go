package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/logrusorgru/aurora/v4"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"github.com/99designs/gqlgen/graphql"
)

type Tracer struct {
	DisableColor bool
	au           *aurora.Aurora
	out          io.Writer
}

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
} = &Tracer{}

func (a Tracer) ExtensionName() string {
	return "ApolloTracing"
}

func (a *Tracer) Validate(schema graphql.ExecutableSchema) error {
	isTTY := isatty.IsTerminal(os.Stdout.Fd())

	a.au = aurora.New(aurora.WithColors(!a.DisableColor && isTTY))
	a.out = colorable.NewColorableStdout()

	return nil
}

func stringify(value any) string {
	valueJson, err := json.MarshalIndent(value, "  ", "  ")
	if err == nil {
		return string(valueJson)
	}

	return fmt.Sprint(value)
}

func (a Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	opCtx := graphql.GetOperationContext(ctx)

	_, _ = fmt.Fprintln(a.out, "GraphQL Request {")
	for _, line := range strings.Split(opCtx.RawQuery, "\n") {
		_, _ = fmt.Fprintln(a.out, " ", aurora.Cyan(line))
	}
	for name, value := range opCtx.Variables {
		_, _ = fmt.Fprintf(a.out, "  var %s = %s\n", name, aurora.Yellow(stringify(value)))
	}
	resp := next(ctx)

	_, _ = fmt.Fprintln(a.out, "  resp:", aurora.Green(stringify(resp)))
	if resp != nil {
		for _, err := range resp.Errors {
			_, _ = fmt.Fprintln(a.out, "  error:", aurora.Bold(err.Path.String()+":"), aurora.Red(err.Message))
		}
	}

	_, _ = fmt.Fprintln(a.out, "}")
	_, _ = fmt.Fprintln(a.out)
	return resp
}
