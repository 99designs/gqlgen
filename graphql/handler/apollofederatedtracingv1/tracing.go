package apollofederatedtracingv1

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/protobuf/proto"
)

type (
	Tracer struct {
		ClientName  string
		Version     string
		Hostname    string
		TreeBuilder *TreeBuilder
		ShouldTrace bool
	}
)

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
	graphql.FieldInterceptor
	graphql.OperationInterceptor
	graphql.OperationParameterMutator
} = &Tracer{}

// ExtensionName returns the name of the extension
func (Tracer) ExtensionName() string {
	return "ApolloFederatedTracingV1"
}

// Validate returns errors based on the schema; since this extension doesn't require validation, we return nil
func (Tracer) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (t *Tracer) MutateOperationParameters(ctx context.Context, request *graphql.RawParams) *gqlerror.Error {
	t.ShouldTrace = request.Headers.Get("apollo-federation-include-trace") == "ftv1" // check for header
	return nil
}

// InterceptOperation acts on each Graph operation; on each operation, start a tree builder and start the tree's timer for tracing
func (t *Tracer) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	if !t.ShouldTrace {
		return next(ctx)
	}

	t.TreeBuilder = NewTreeBuilder()

	return next(ctx)
}

// InterceptField is called on each field's resolution, including information about the path and parent node.
// This information is then used to build the relevant Node Tree used in the FTV1 tracing format
func (t *Tracer) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	if !t.ShouldTrace {
		return next(ctx)
	}

	t.TreeBuilder.WillResolveField(ctx)

	return next(ctx)
}

// InterceptResponse is called before the overall response is sent, but before each field resolves; as a result
// the final marshaling is deferred to happen at the end of the operation
func (t *Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	if !t.ShouldTrace {
		return next(ctx)
	}
	t.TreeBuilder.StartTimer(ctx)

	// because we need to update the ftv1 string at a later time (as fields resolve before the response is sent),
	// we instantiate the string and use a pointer to be able to update later
	var ftv1 string
	graphql.RegisterExtension(ctx, "ftv1", &ftv1)

	// now that fields have finished resolving, it stops the timer to calculate trace duration
	defer func() {
		t.TreeBuilder.StopTimer(ctx)

		// marshal the protobuf ...
		p, err := proto.Marshal(t.TreeBuilder.Trace)
		if err != nil {
			fmt.Print(err)
		}

		// ... then set the previously instantiated string as the base64 formatted string as required
		ftv1 = base64.StdEncoding.EncodeToString(p)
	}()

	resp := next(ctx)
	return resp
}
