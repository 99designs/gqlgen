package execbehavior

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

func TestFieldContextChild(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ChildProbe = func(ctx context.Context) (*ChildProbe, error) {
		return &ChildProbe{}, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	var (
		mu      sync.Mutex
		results = map[string]string{}
	)
	describe := func(child *graphql.FieldContext, err error) string {
		if err != nil {
			return "error: " + err.Error()
		}
		args := map[string]any{}
		for k, v := range child.Args {
			if p, ok := v.(*int); ok && p != nil {
				v = *p
			}
			args[k] = v
		}
		return fmt.Sprintf("%s.%s method=%t resolver=%t args=%v",
			child.Object, child.Field.Name, child.IsMethod, child.IsResolver, args)
	}
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		if fc.Object != "ChildProbe" {
			return next(ctx)
		}
		var found []string
		for _, sel := range graphql.CollectFields(graphql.GetOperationContext(ctx), fc.Field.Selections, []string{"ChildProbeObject"}) {
			found = append(found, describe(fc.Child(ctx, sel)))
		}
		missing := graphql.CollectedField{Field: &ast.Field{Name: "missing", Alias: "missing"}}
		found = append(found, describe(fc.Child(ctx, missing)))
		mu.Lock()
		results[fc.Field.Name] = fmt.Sprint(found)
		mu.Unlock()
		return next(ctx)
	})

	got := post(t, srv, `{ childProbe {
		scalar enumValue
		union { __typename }
		iface { id }
		object { id nested nestedWithArg: nested(limit: 5) }
		list { id }
	} }`)
	require.JSONEq(t, `{"data":{"childProbe":{"scalar":null,"enumValue":null,"union":null,`+
		`"iface":null,"object":null,"list":null}}}`, got)

	require.Equal(t, map[string]string{
		"scalar":    "[error: field of type String does not have child fields]",
		"enumValue": "[error: field of type ProbeEnum does not have child fields]",
		"union":     "[error: field of type ProbeUnion does not have child fields error: field of type ProbeUnion does not have child fields]",
		"iface":     "[error: FieldContext.Child cannot be called on type INTERFACE error: FieldContext.Child cannot be called on type INTERFACE]",
		"object": "[ChildProbeObject.id method=false resolver=false args=map[] " +
			"ChildProbeObject.nested method=false resolver=false args=map[limit:3] " +
			"ChildProbeObject.nested method=false resolver=false args=map[limit:5] " +
			`error: no field named "missing" was found under type ChildProbeObject]`,
		"list": "[ChildProbeObject.id method=false resolver=false args=map[] " +
			`error: no field named "missing" was found under type ChildProbeObject]`,
	}, results)
}
