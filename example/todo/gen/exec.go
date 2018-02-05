package gen

import (
	"bytes"
	"fmt"

	"github.com/vektah/graphql-go/errors"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/relay"
	"github.com/vektah/graphql-go/validation"
)

func NewResolver(resolvers Resolvers) relay.Resolver {
	return func(document string, operationName string, variables map[string]interface{}) *jsonw.Response {
		doc, qErr := query.Parse(document)
		if qErr != nil {
			return &jsonw.Response{Errors: []*errors.QueryError{qErr}}
		}

		errs := validation.Validate(parsedSchema, doc)
		if len(errs) != 0 {
			return &jsonw.Response{Errors: errs}
		}

		op, err := doc.GetOperation(operationName)
		if err != nil {
			return &jsonw.Response{Errors: []*errors.QueryError{errors.Errorf("%s", err)}}
		}

		c := executionContext{
			resolvers: resolvers,
			variables: variables,
			doc:       doc,
		}

		// TODO: parallelize if query.

		var data jsonw.Encodable
		if op.Type == query.Query {
			data = queryType{}.executeSelectionSet(&c, op.Selections, nil)
		} else if op.Type == query.Mutation {
			data = mutationType{}.executeSelectionSet(&c, op.Selections, nil)
		} else {
			return &jsonw.Response{Errors: []*errors.QueryError{errors.Errorf("unsupported operation type")}}
		}

		b := &bytes.Buffer{}
		data.JSON(b)
		return &jsonw.Response{
			Data:   b.Bytes(),
			Errors: c.Errors,
		}
	}
}

type executionContext struct {
	errors.Builder
	resolvers Resolvers
	variables map[string]interface{}
	doc       *query.Document
}

type resolvedType interface {
	accepts(name string) bool
}

func (c *executionContext) introspectSchema() *introspection.Schema {
	return introspection.WrapSchema(parsedSchema)
}

func (c *executionContext) introspectType(name string) *introspection.Type {
	t := parsedSchema.Resolve(name)
	if t == nil {
		return nil
	}
	return introspection.WrapType(t)
}

func (c *executionContext) collectFields(selSet []query.Selection, resolver resolvedType, visited map[string]bool) []collectedField {
	var groupedFields []collectedField

	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *query.Field:
			f := getOrCreateField(&groupedFields, sel.Name.Name, func() collectedField {
				f := collectedField{
					Alias: sel.Alias.Name,
					Name:  sel.Name.Name,
				}
				if len(sel.Arguments) > 0 {
					f.Args = map[string]interface{}{}
					for _, arg := range sel.Arguments {
						f.Args[arg.Name.Name] = arg.Value.Value(c.variables)
					}
				}
				return f
			})

			f.Selections = append(f.Selections, sel.Selections...)
		case *query.InlineFragment:
			if !resolver.accepts(sel.On.Ident.Name) {
				continue
			}

			for _, childField := range c.collectFields(sel.Selections, resolver, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() collectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		case *query.FragmentSpread:
			fragmentName := sel.Name.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			visited[fragmentName] = true

			fragment := c.doc.Fragments.Get(fragmentName)
			if fragment == nil {
				c.Errorf("missing fragment %s", fragmentName)
				continue
			}

			if !resolver.accepts(fragment.On.Name) {
				continue
			}

			for _, childField := range c.collectFields(fragment.Selections, resolver, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() collectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

type collectedField struct {
	Alias      string
	Name       string
	Args       map[string]interface{}
	Selections []query.Selection
}

func getOrCreateField(c *[]collectedField, name string, creator func() collectedField) *collectedField {
	for i, cf := range *c {
		if cf.Alias == name {
			return &(*c)[i]
		}
	}

	f := creator()

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}
