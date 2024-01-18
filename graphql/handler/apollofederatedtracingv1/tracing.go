package apollofederatedtracingv1

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/99designs/gqlgen/graphql"
)

type (
	Tracer struct {
		ClientName string
		Version    string
		Hostname   string
	}

	treeBuilderKey string
)

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
	return next(context.WithValue(ctx, key, NewTreeBuilder()))
}

// InterceptField is called on each field's resolution, including information about the path and parent node.
// This information is then used to build the relevant Node Tree used in the FTV1 tracing format
func (t *Tracer) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
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
