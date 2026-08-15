package introspection

import (
	"sort"

	"github.com/vektah/gqlparser/v2/ast"
)

type Schema struct {
	schema *ast.Schema
}

func (s *Schema) Description() *string {
	if s.schema.Description == "" {
		return nil
	}
	return &s.schema.Description
}

func (s *Schema) Types() []Type {
	defs := make([]*ast.Definition, 0, len(s.schema.Types))
	for _, typ := range s.schema.Types {
		defs = append(defs, typ)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

	types := make([]Type, len(defs))
	for i, def := range defs {
		types[i] = *WrapTypeFromDef(s.schema, def)
	}
	return types
}

func (s *Schema) QueryType() *Type {
	return WrapTypeFromDef(s.schema, s.schema.Query)
}

func (s *Schema) MutationType() *Type {
	return WrapTypeFromDef(s.schema, s.schema.Mutation)
}

func (s *Schema) SubscriptionType() *Type {
	return WrapTypeFromDef(s.schema, s.schema.Subscription)
}

func (s *Schema) Directives() []Directive {
	defs := make([]*ast.DirectiveDefinition, 0, len(s.schema.Directives))
	for _, d := range s.schema.Directives {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

	res := make([]Directive, len(defs))
	for i, d := range defs {
		res[i] = s.directiveFromDef(d)
	}
	return res
}

func (s *Schema) directiveFromDef(d *ast.DirectiveDefinition) Directive {
	locs := make([]string, len(d.Locations))
	for i, loc := range d.Locations {
		locs[i] = string(loc)
	}

	args := make([]InputValue, len(d.Arguments))
	for i, arg := range d.Arguments {
		args[i] = InputValue{
			Name:         arg.Name,
			description:  arg.Description,
			DefaultValue: defaultValue(arg.DefaultValue),
			Type:         WrapTypeFromType(s.schema, arg.Type),
		}
	}

	return Directive{
		Name:         d.Name,
		description:  d.Description,
		Locations:    locs,
		Args:         args,
		IsRepeatable: d.IsRepeatable,
	}
}
