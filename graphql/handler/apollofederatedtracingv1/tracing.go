package apollofederatedtracingv1

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/99designs/gqlgen/graphql"
	tracing_logger "github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/logger"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	ERROR_MASKED     = "masked"
	ERROR_UNMODIFIED = "all"
	ERROR_TRANSFORM  = "transform"
)

type (
	Tracer struct {
		ClientName   string
		Version      string
		Hostname     string
		ErrorOptions *ErrorOptions

		// Logger is used to log errors that occur during the tracing process; if nil, no logging will occur
		// This can use the default Go logger or a custom logger (e.g. logrus or zap)
		Logger tracing_logger.Logger
	}

	treeBuilderKey string
)

type ErrorOptions struct {
	// ErrorOptions is the option to handle errors in the trace, it can be one of the following:
	// - "masked": masks all errors
	// - "all": includes all errors
	// - "transform": includes all errors but transforms them using TransformFunction, which can allow users to redact sensitive information
	ErrorOption       string
	TransformFunction func(g *gqlerror.Error) *gqlerror.Error
}

const (
	key = treeBuilderKey("treeBuilder")
)

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
	graphql.FieldInterceptor
	graphql.OperationInterceptor
} = &Tracer{}

// ExtensionName returns the name of the extension
func (Tracer) ExtensionName() string {
	return "ApolloFederatedTracingV1"
}

// Validate returns errors based on the schema; since this extension doesn't require validation, we return nil
func (Tracer) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (t *Tracer) shouldTrace(ctx context.Context) bool {
	return graphql.HasOperationContext(ctx) &&
		graphql.GetOperationContext(ctx).Headers.Get("apollo-federation-include-trace") == "ftv1"
}

func (t *Tracer) getTreeBuilder(ctx context.Context) *TreeBuilder {
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	if tb, ok := val.(*TreeBuilder); ok {
		return tb
	}
	return nil
}

// InterceptOperation acts on each Graph operation; on each operation, start a tree builder and start the tree's timer for tracing
func (t *Tracer) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	if !t.shouldTrace(ctx) {
		return next(ctx)
	}

	return next(context.WithValue(ctx, key, NewTreeBuilder(t.ErrorOptions, t.Logger)))
}

// InterceptField is called on each field's resolution, including information about the path and parent node.
// This information is then used to build the relevant Node Tree used in the FTV1 tracing format
func (t *Tracer) InterceptField(ctx context.Context, next graphql.Resolver) (any, error) {
	if !t.shouldTrace(ctx) {
		return next(ctx)
	}
	if tb := t.getTreeBuilder(ctx); tb != nil {
		tb.WillResolveField(ctx)
	}

	return next(ctx)
}

// InterceptResponse is called before the overall response is sent, but before each field resolves; as a result
// the final marshaling is deferred to happen at the end of the operation
func (t *Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	if !t.shouldTrace(ctx) {
		return next(ctx)
	}
	tb := t.getTreeBuilder(ctx)
	if tb == nil {
		return next(ctx)
	}

	tb.StartTimer(ctx)

	val := new(string)
	graphql.RegisterExtension(ctx, "ftv1", val)

	// now that fields have finished resolving, it stops the timer to calculate trace duration
	defer func(val *string) {
		errors := graphql.GetErrors(ctx)
		if len(errors) > 0 {
			tb.DidEncounterErrors(ctx, errors)
		}

		tb.StopTimer(ctx)
		// marshal the protobuf ...
		p, err := proto.Marshal(tb.Trace)
		if err != nil {
			fmt.Print(err)
		}

		// ... then set the previously instantiated string as the base64 formatted string as required
		*val = base64.StdEncoding.EncodeToString(p)
	}(val)
	resp := next(ctx)
	return resp
}
