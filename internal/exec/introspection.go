package exec

import (
	"context"
	"reflect"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
	"github.com/neelance/graphql-go/trace"
)

var schemaExec *objectExec
var typeExec *objectExec

func init() {
	var err error
	b := newExecBuilder(schema.Meta)

	metaSchema := schema.Meta.Types["__Schema"].(*schema.Object)
	schemaExec, err = b.makeObjectExec(metaSchema.Name, metaSchema.Fields, nil, false, reflect.TypeOf(&introspection.Schema{}))
	if err != nil {
		panic(err)
	}

	metaType := schema.Meta.Types["__Type"].(*schema.Object)
	typeExec, err = b.makeObjectExec(metaType.Name, metaType.Fields, nil, false, reflect.TypeOf(&introspection.Type{}))
	if err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}
}

func IntrospectSchema(s *schema.Schema) interface{} {
	r := &Request{
		Schema:  s,
		Doc:     introspectionQuery,
		Limiter: make(chan struct{}, 10),
		Tracer:  trace.NoopTracer{},
	}
	sels := applySelectionSet(r, schemaExec, introspectionQuery.Operations.Get("IntrospectionQuery").SelSet)
	resolver := reflect.ValueOf(introspection.WrapSchema(r.Schema))
	results := make(map[string]interface{})
	for _, sel := range sels {
		execSelection(context.Background(), sel, resolver, results)
	}
	return results
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
