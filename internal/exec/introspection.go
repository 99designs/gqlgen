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
	b := newExecBuilder(schema.Meta)

	if err := b.assignExec(&schemaExec, schema.Meta.Types["__Schema"], reflect.TypeOf(&introspection.Schema{})); err != nil {
		panic(err)
	}

	if err := b.assignExec(&typeExec, schema.Meta.Types["__Type"], reflect.TypeOf(&introspection.Type{})); err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}
}

func IntrospectSchema(s *schema.Schema) (interface{}, error) {
	r := &Request{
		Schema:  s,
		Doc:     introspectionQuery,
		Limiter: make(chan struct{}, 10),
	}
	return introspectSchema(context.Background(), r, introspectionQuery.Operations.Get("IntrospectionQuery").SelSet), nil
}

func introspectSchema(ctx context.Context, r *Request, selSet *query.SelectionSet) interface{} {
	return schemaExec.exec(ctx, r, selSet, reflect.ValueOf(introspection.WrapSchema(r.Schema)))
}

func introspectType(ctx context.Context, r *Request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.Schema.Types[name]
	if !ok {
		return nil
	}
	return typeExec.exec(ctx, r, selSet, reflect.ValueOf(introspection.WrapType(t)))
}

var introspectionQuery *query.Document

func init() {
	var err *errors.QueryError
	introspectionQuery, err = query.Parse(introspectionQuerySrc)
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
