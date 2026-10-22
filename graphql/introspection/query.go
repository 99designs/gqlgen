package introspection

// QueryOptions selects introspection fields that not every server supports.
// The zero value requests only fields that graphql-js has supported since
// 16.0.0. The options match graphql-js getIntrospectionQuery's options of the
// same names. A client that doesn't know the server can first query
// __type(name: "__Type") { fields { name } } and enable only the fields listed.
type QueryOptions struct {
	// OneOf requests __Type.isOneOf. graphql-js added it in 16.9.0 and
	// 17.0.0-alpha.3, so 16.0-16.8 and 17.0.0-alpha.1 and alpha.2 reject the
	// query. gqlgen servers always support it.
	OneOf bool
}

// Query is a full introspection query: it requests every field the gqlparser
// prelude declares on the introspection types, including deprecated fields,
// arguments and input fields, except the optional fields QueryOptions
// selects. It equals NewQuery(QueryOptions{}).
const Query = queryBeforeTypeOptions + queryAfterTypeOptions

// NewQuery returns Query plus the optional fields opts selects.
func NewQuery(opts QueryOptions) string {
	var typeOptions string
	if opts.OneOf {
		typeOptions += "  isOneOf\n"
	}
	return queryBeforeTypeOptions + typeOptions + queryAfterTypeOptions
}

// queryBeforeTypeOptions and queryAfterTypeOptions are Query split where
// NewQuery adds the optional fields of the FullType fragment.
const (
	queryBeforeTypeOptions = `
query IntrospectionQuery {
  __schema {
    description
    queryType {
      name
    }
    mutationType {
      name
    }
    subscriptionType {
      name
    }
    types {
      ...FullType
    }
    directives {
      name
      description
      isRepeatable
      locations
      args(includeDeprecated: true) {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  specifiedByURL
`
	queryAfterTypeOptions = `  fields(includeDeprecated: true) {
    name
    description
    args(includeDeprecated: true) {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields(includeDeprecated: true) {
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
  type {
    ...TypeRef
  }
  defaultValue
  isDeprecated
  deprecationReason
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
  }
}
`
)
