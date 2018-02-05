package gen

import (
	"context"
	"fmt"
	"io"

	"github.com/mitchellh/mapstructure"
	"github.com/vektah/graphql-go/errors"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/relay"
	"github.com/vektah/graphql-go/validation"
)

func NewResolver(resolvers Resolvers) relay.Resolver {
	return func(ctx context.Context, document string, operationName string, variables map[string]interface{}, w io.Writer) []*errors.QueryError {
		doc, qErr := query.Parse(document)
		if qErr != nil {
			return []*errors.QueryError{qErr}
		}

		errs := validation.Validate(parsedSchema, doc)
		if len(errs) != 0 {
			return errs
		}

		op, err := doc.GetOperation(operationName)
		if err != nil {
			return []*errors.QueryError{errors.Errorf("%s", err)}
		}

		if op.Type != query.Query && op.Type != query.Mutation {
			return []*errors.QueryError{errors.Errorf("unsupported operation type")}
		}

		c := executionContext{
			resolvers: resolvers,
			variables: variables,
			doc:       doc,
			ctx:       ctx,
			json:      jsonw.New(w),
		}

		// TODO: parallelize if query.

		c.json.BeginObject()

		c.json.ObjectKey("data")

		if op.Type == query.Query {
			_query(&c, op.Selections, nil)
		} else if op.Type == query.Mutation {
			_mutation(&c, op.Selections, nil)
		} else {
			c.Errorf("unsupported operation %s", op.Type)
			c.json.Null()
		}

		if len(c.Errors) > 0 {
			c.json.ObjectKey("errors")
			errors.WriteErrors(w, c.Errors)
		}

		c.json.EndObject()
		return nil
	}
}

type executionContext struct {
	errors.Builder
	json      *jsonw.Writer
	resolvers Resolvers
	variables map[string]interface{}
	doc       *query.Document
	ctx       context.Context
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

func instanceOf(val string, satisfies []string) bool {
	for _, s := range satisfies {
		if val == s {
			return true
		}
	}
	return false
}

func (c *executionContext) collectFields(selSet []query.Selection, satisfies []string, visited map[string]bool) []collectedField {
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
			if !instanceOf(sel.On.Ident.Name, satisfies) {
				continue
			}

			for _, childField := range c.collectFields(sel.Selections, satisfies, visited) {
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

			if !instanceOf(fragment.On.Ident.Name, satisfies) {
				continue
			}

			for _, childField := range c.collectFields(fragment.Selections, satisfies, visited) {
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

func unpackComplexArg(result interface{}, data interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "graphql",
		ErrorUnused: true,
		Result:      result,
	})
	if err != nil {
		panic(err)
	}

	return decoder.Decode(data)
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
