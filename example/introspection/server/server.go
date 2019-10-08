package main

import (
	"context"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/example/introspection"
	"github.com/vektah/gqlparser/ast"

	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Introspection Auth", "/query"))
	exec := introspection.NewExecutableSchema(introspection.Config{
		Resolvers: nil,
		Directives: introspection.DirectiveRoot{
			Introspection: introspection.IntrospectionDirective(introspection.IntrospectionConfig{
				HideFunc: func(ctx context.Context, directive *ast.Directive) (bool, error) {
					return false, nil
				},
				RequireAuthFunc: func(ctx context.Context, directive *ast.Directive) (bool, error) {
					// extract auth from context
					return false, nil
				},
			}),
		},
		Complexity: introspection.ComplexityRoot{},
	})
	http.Handle("/query", handler.GraphQL(exec))
	log.Print("connect to http://localhost:8080/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
