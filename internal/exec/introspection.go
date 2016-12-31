package exec

import (
	"context"
	"reflect"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
)

var schemaExec iExec
var typeExec iExec

func init() {
	{
		var err error
		schemaExec, err = makeWithType(schema.Meta, schema.Meta.Types["__Schema"], &introspection.Schema{})
		if err != nil {
			panic(err)
		}
	}

	{
		var err error
		typeExec, err = makeWithType(schema.Meta, schema.Meta.Types["__Type"], &introspection.Type{})
		if err != nil {
			panic(err)
		}
	}
}

func IntrospectSchema(s *schema.Schema) (interface{}, error) {
	r := &request{
		schema: s,
		doc:    introspectionQuery,
	}
	return introspectSchema(context.Background(), r, introspectionQuery.Operations["IntrospectionQuery"].SelSet), nil
}

func introspectSchema(ctx context.Context, r *request, selSet *query.SelectionSet) interface{} {
	return schemaExec.exec(ctx, r, selSet, reflect.ValueOf(&introspection.Schema{Schema: r.schema}), false)
}

func introspectType(ctx context.Context, r *request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.schema.Types[name]
	if !ok {
		return nil
	}
	return typeExec.exec(ctx, r, selSet, reflect.ValueOf(&introspection.Type{t}), false)
}

var introspectionQuery *query.Document

func init() {
	var err *errors.QueryError
	introspectionQuery, err = query.Parse(introspectionQuerySrc, schema.Meta.Resolve)
	if err != nil {
		panic(err)
	}
}

var introspectionQuerySrc = `
  query IntrospectionQuery {
    __schema {
      queryType { name }
      mutationType { name }
      subscriptionType { name }
      types {
        ...FullType
      }
      directives {
        name
        description
        locations
        args {
          ...InputValue
        }
      }
    }
  }
  fragment FullType on __Type {
    kind
    name
    description
    fields(includeDeprecated: true) {
      name
      description
      args {
        ...InputValue
      }
      type {
        ...TypeRef
      }
      isDeprecated
      deprecationReason
    }
    inputFields {
      ...InputValue
    }
    interfaces {
      ...TypeRef
    }
    enumValues(includeDeprecated: true) {
      name
      description
      isDeprecated
      deprecationReason
    }
    possibleTypes {
      ...TypeRef
    }
  }
  fragment InputValue on __InputValue {
    name
    description
    type { ...TypeRef }
    defaultValue
  }
  fragment TypeRef on __Type {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
        }
      }
    }
  }
`
