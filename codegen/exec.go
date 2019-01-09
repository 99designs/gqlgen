package codegen

import (
	"fmt"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/codegen/unified"
)

type ExecBuild struct {
	*unified.Schema

	PackageName      string
	QueryRoot        *unified.Object
	MutationRoot     *unified.Object
	SubscriptionRoot *unified.Object
}

// bind a schema together with some code to generate a Build
func buildExec(s *unified.Schema) error {
	b := &ExecBuild{
		Schema:      s,
		PackageName: s.Config.Exec.Package,
	}

	if s.Schema.Query != nil {
		b.QueryRoot = b.Objects.ByName(s.Schema.Query.Name)
	} else {
		return fmt.Errorf("query entry point missing")
	}

	if s.Schema.Mutation != nil {
		b.MutationRoot = b.Objects.ByName(s.Schema.Mutation.Name)
	}

	if s.Schema.Subscription != nil {
		b.SubscriptionRoot = b.Objects.ByName(s.Schema.Subscription.Name)
	}

	return templates.RenderToFile("generated.gotpl", s.Config.Exec.Filename, b)

}
