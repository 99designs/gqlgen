package codegen

import (
	// Import and ignore the ambient imports listed below so dependency managers
	// don't prune unused code for us. Both lists should be kept in sync.
	_ "github.com/monzo/gqlgen/graphql"
	_ "github.com/monzo/gqlgen/graphql/introspection"
	_ "github.com/vektah/gqlparser"
	_ "github.com/vektah/gqlparser/ast"
)
