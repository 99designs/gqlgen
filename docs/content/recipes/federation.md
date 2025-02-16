---
title: 'Using Apollo federation gqlgen'
description: How federate many services into a single graph using Apollo
linkTitle: Apollo Federation
menu: { main: { parent: 'recipes' } }
---

In this quick guide we are going to implement the example [Apollo Federation](https://www.apollographql.com/docs/apollo-server/federation/introduction/)
server in gqlgen. You can find the finished result in the [examples directory](https://github.com/99designs/gqlgen/tree/master/_examples/federation).

## Enable federation

Uncomment federation configuration in your `gqlgen.yml`

```yml
# Uncomment to enable federation
federation:
  filename: graph/federation.go
  package: graph
```

### Federation 2

If you are using Apollo's Federation 2 standard, your schema should automatically be upgraded so long as you include the required `@link` directive within your schema. If you want to force Federation 2 composition, the `federation` configuration supports a `version` flag to override that. For example:

```yml
federation:
  filename: graph/federation.go
  package: graph
  version: 2
```

## Create the federated servers

For each server to be federated we will create a new gqlgen project.

```bash
go run github.com/99designs/gqlgen
```

Update the schema to reflect the federated example
```graphql
type Review {
  body: String
  author: User @provides(fields: "username")
  product: Product
}

extend type User @key(fields: "id") {
  id: ID! @external # External directive not required for key fields in federation v2
  reviews: [Review]
}

extend type Product @key(fields: "upc") {
  upc: String! @external # External directive not required for key fields in federation v2
  reviews: [Review]
}
```


and regenerate
```bash
go run github.com/99designs/gqlgen
```

then implement the resolvers
```go
// These two methods are required for gqlgen to resolve the internal id-only wrapper structs.
// This boilerplate might be removed in a future version of gqlgen that can no-op id only nodes.
func (r *entityResolver) FindProductByUpc(ctx context.Context, upc string) (*model.Product, error) {
	return &model.Product{
		Upc: upc,
	}, nil
}

func (r *entityResolver) FindUserByID(ctx context.Context, id string) (*model.User, error) {
	return &model.User{
		ID: id,
	}, nil
}

// Here we implement the stitched part of this service, returning reviews for a product. Of course normally you would
// go back to the database, but we are just making some data up here.
func (r *productResolver) Reviews(ctx context.Context, obj *model.Product) ([]*model.Review, error) {
	switch obj.Upc {
	case "top-1":
		return []*model.Review{{
			Body: "A highly effective form of birth control.",
		}}, nil

	case "top-2":
		return []*model.Review{{
			Body: "Fedoras are one of the most fashionable hats around and can look great with a variety of outfits.",
		}}, nil

	case "top-3":
		return []*model.Review{{
			Body: "This is the last straw. Hat you will wear. 11/10",
		}}, nil

	}
	return nil, nil
}

func (r *userResolver) Reviews(ctx context.Context, obj *model.User) ([]*model.Review, error) {
	if obj.ID == "1234" {
		return []*model.Review{{
			Body: "Has an odd fascination with hats.",
		}}, nil
	}
	return nil, nil
}
```

> Note
>
> Repeat this step for each of the services in the apollo doc (accounts, products, reviews)

## Create the federation gateway

```bash
npm install --save @apollo/gateway apollo-server graphql
```

```typescript
const { ApolloServer } = require('apollo-server');
const { ApolloGateway, IntrospectAndCompose } = require("@apollo/gateway");

const gateway = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            { name: 'accounts', url: 'http://localhost:4001/query' },
            { name: 'products', url: 'http://localhost:4002/query' },
            { name: 'reviews', url: 'http://localhost:4003/query' }
        ]
    })
});

const server = new ApolloServer({
    gateway,

    subscriptions: false,
});

server.listen().then(({ url }) => {
    console.log(`🚀 Server ready at ${url}`);
});
```

## Start all the services

In separate terminals:
```bash
go run accounts/server.go
go run products/server.go
go run reviews/server.go
node gateway/index.js
```

## Query the federated gateway

The examples from the apollo doc should all work, eg

```graphql
query {
  me {
    username
    reviews {
      body
      product {
        name
        upc
      }
    }
  }
}
```

should return

```json
{
  "data": {
    "me": {
      "username": "Me",
      "reviews": [
        {
          "body": "A highly effective form of birth control.",
          "product": {
            "name": "Trilby",
            "upc": "top-1"
          }
        },
        {
          "body": "Fedoras are one of the most fashionable hats around and can look great with a variety of outfits.",
          "product": {
            "name": "Trilby",
            "upc": "top-1"
          }
        }
      ]
    }
  }
}
```

## Using @requires

`@requires` enables you to [define computed fields](https://www.apollographql.com/docs/federation/federated-schemas/federated-directives/#requires). In order for this to work, you need to be able to reference the values injected by the selection set inside the `fields` property of `@requires`.

In order to do this, you need to enable the `federation.options.computed_requires` flag. You also
need to enable `call_argument_directives_with_null`.

```yml
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
    computed_requires: true

call_argument_directives_with_null: true
```

Once you do this, if you have `@requires` declared anywhere on your schema, you'll see updates to the
genrated resolver functions that include a new argument, `federationRequires`, that will contain the
fields you requested in your `@requires.fields` selection set.

> Note: currently it's represented as a map[string]any where the contained values are encoded with
`encoding/json`. Eventually we will generate a typesafe model that represents these models,
however that is a large lift. This typesafe support will be added in the future.

### Example

Take a simple todo app schema that needs to provide a formatted status text to be used across all clients by referencing the assignee's name.

```graphql
type Todo @key(fields:"id") {
  id: ID!
  text: String!
  statusText: String! @requires(fields: "assignee { name }")
  status: String!
  owner: User!
  assignee: User! @external
}

type User @key(fields:"id") {
  id: ID!
  name: String! @external
}
```

The `statusText` resolver function is updated and can be modified accordingly to use the todo representation with the assignee name.

```golang
func (r *todoResolver) StatusText(ctx context.Context, entity *model.Todo, federationRequires map[string]interface{} /* new argument generated onto your resolver function */) (string, error) {
  if federationRequires["assignee"] == nil {
    return "", nil
  }

  // federationRequires will contain the "assignee.name" field provided by the Federation router
  statusText := entity.Status + " by " + federationRequires["assignee"].(map[string]interface{})["name"].(string)
  return statusText, nil
}
```

### [DEPRECATED] Alternate API

> Note: it's not recommended to use this API anymore. See the `Using @requires` section for the recommend API.

If you need to support **nested** or **array** fields in the `@requires` directive, this can be enabled in the configuration by setting `federation.options.explicit_requires` to true.

```yml
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
    explicit_requires: true
```

Enabling this will generate corresponding functions with the entity representations received in the request. This allows for the entity model to be explicitly populated with the required data provided.

#### Example

Take a simple todo app schema that needs to provide a formatted status text to be used across all clients by referencing the assignee's name.

```graphql
type Todo @key(fields:"id") {
  id: ID!
  text: String!
  statusText: String! @requires(fields: "assignee { name }")
  status: String!
  owner: User!
  assignee: User!
}

type User @key(fields:"id") {
  id: ID!
  name: String! @external
}
```

A `PopulateTodoRequires` function is generated, and can be modified accordingly to use the todo representation with the assignee name.

```golang
// PopulateTodoRequires is the requires populator for the Todo entity.
func (ec *executionContext) PopulateTodoRequires(ctx context.Context, entity *model.Todo, reps map[string]interface{}) error {
	if reps["assignee"] != nil {
		entity.StatusText = entity.Status + " by " + reps["assignee"].(map[string]interface{})["name"].(string)
	}
	return nil
}
```
