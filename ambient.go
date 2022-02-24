package main

// Ambient imports
//
// These imports are referenced by the generated code.
// Dependency managers like dep have been known to prune them,
// so explicitly import them here.
//
// TODO: check if these imports are still necessary with go
// modules, and if so move the ambient import to the files
// where they are actually referenced
import (
	_ "github.com/99designs/gqlgen/graphql"
	_ "github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/99designs/gqlgen/graphql/introspection"
	_ "github.com/99designs/gqlgen/handler"
	_ "github.com/vektah/gqlparser/v2"
	_ "github.com/vektah/gqlparser/v2/ast"
)
