---
title: "Using Apollo federation gqlgen"
description: How federate many services into a single graph using Apollo
linkTitle: Apollo Federation
menu: { main: { parent: "recipes" } }
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
go tool gqlgen generate
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
go tool gqlgen generate
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
const { ApolloServer } = require("apollo-server");
const { ApolloGateway, IntrospectAndCompose } = require("@apollo/gateway");

const gateway = new ApolloGateway({
	supergraphSdl: new IntrospectAndCompose({
		subgraphs: [
			{ name: "accounts", url: "http://localhost:4001/query" },
			{ name: "products", url: "http://localhost:4002/query" },
			{ name: "reviews", url: "http://localhost:4003/query" },
		],
	}),
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

gqlgen offers several mutually-exclusive strategies for delivering `@requires`
data to your code. They all answer the same question — *how does the required
data reach the resolver?* — so each entity uses **exactly one**:

| Strategy | Package option | `requires:` value | How `@requires` data is delivered |
| --- | --- | --- | --- |
| Default | _(none)_ | `"default"` | Unmarshaled onto the returned entity, after the resolver runs. |
| Explicit | `explicit_requires` | `"explicit"` | A `Populate<Entity>Requires` function you implement, called on the returned entity after the resolver. Supports nested/array fields. |
| Computed | `computed_requires` | `@computedRequires` (per field) | Delivered to standalone field resolvers via a `federationRequires` argument (Federation 2 only). |
| Preloaded | `preloaded_requires` | `"preloaded"` | Unmarshaled onto the resolver's *input* representation, before the resolver runs, so a multi resolver sees every entity's `@requires` data at once. Flat scalar/enum fields only; requires `multi`. |

The package option sets the **default** for the whole subgraph. The `requires:`
argument on `@entityResolver` overrides that default **per entity**, choosing
among `default`, `explicit`, and `preloaded`. (`computed` is not a `requires:`
value — it describes field-resolver delivery rather than how data reaches the
entity resolver, so it is selected by the `computed_requires` package option or,
per field, by `@computedRequires`.) Because `@entityResolver` is your own directive, add
the `requires: String` argument to its definition. With no `requires:` argument,
an entity falls back to the package default.

#### Computing individual `@requires` fields with `@computedRequires`

`computed_requires` computes **every** `@requires` field on its entities. To
compute just **one** field, annotate it with `@computedRequires` (a gqlgen-provided
directive; no declaration needed). That field is delivered to a standalone field
resolver, while the entity's other `@requires` fields follow its strategy. This
is what lets a `preloaded` entity keep its scalar `@requires` on the batch input
*and* handle an object-typed `@requires` — which `preloaded` cannot reconstruct —
on the same entity:

```graphql
type Product @key(fields: "id") @entityResolver(multi: true) {
  id: ID! @external
  category: String! @external
  info: Info! @external
  display: String! @requires(fields: "category")            # preloaded onto the input
  summary: String! @requires(fields: "info { label }") @computedRequires  # computed field resolver
}
```

`@computedRequires` requires Federation 2 and `call_argument_directives_with_null`. It
only applies to `@requires` fields, and cannot be combined with the `explicit`
strategy (whose `Populate<Entity>Requires` hook already owns every `@requires`
field). See `_examples/multi-entity-tests` for a worked subgraph.

To mix whole-entity strategies instead — for example `computed` for an
object-typed entity and `preloaded` for a scalar one — make `computed` the
package default and override the scalar entity to `preloaded`:

```graphql
directive @entityResolver(multi: Boolean, requires: String) on OBJECT

# with computed_requires: true, computed is the package default
type Planet  @key(fields: "name")                                                     { ... }
type Product @key(fields: "id")   @entityResolver(multi: true, requires: "preloaded") { ... }
```

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
> `encoding/json`. Eventually we will generate a typesafe model that represents these models,
> however that is a large lift. This typesafe support will be added in the future.

### Example

Take a simple todo app schema that needs to provide a formatted status text to be used across all clients by referencing the assignee's name.

```graphql
type Todo @key(fields: "id") {
	id: ID!
	text: String!
	statusText: String! @requires(fields: "assignee { name }")
	status: String!
	owner: User!
	assignee: User! @external
}

type User @key(fields: "id") {
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
type Todo @key(fields: "id") {
	id: ID!
	text: String!
	statusText: String! @requires(fields: "assignee { name }")
	status: String!
	owner: User!
	assignee: User!
}

type User @key(fields: "id") {
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

## Using @entityResolver

The `@entityResolver` directive enables optimization for entity resolver generation in GraphQL federation.

### Configuration

To use this feature, define the `@entityResolver(multi: Boolean)` directive on your OBJECT types. Federated entities must be annotated with this directive to enable the functionality.

Example:

```graphql
type MultiHello @key(fields: "name") @entityResolver(multi: true)
```

### Global Configuration

You can enable this feature by default by setting the `federation.options.entity_resolver_multi` flag in your configuration:

```yml
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
    entity_resolver_multi: true
```

> **Roadmap — the default will change.** A future major version plans to make
> `multi` the **default** (single-entity resolution becomes "multi with N = 1")
> and to change the single-entity resolver signature so it takes the same input
> struct as the multi resolver — `FindProductByID(ctx, rep *model.ProductByIDsInput)`
> instead of positional key arguments (`FindProductByID(ctx, id string)`). To be
> unaffected when the default flips, **declare `multi:` explicitly** on your
> entities now (`multi: true` to opt into batching, `multi: false` to keep
> single-entity resolution); entities that already state their choice see no
> change. See `_examples/multi-entity-tests/PLAN.md` for the sequencing.

### Schema Example

```graphql
directive @entityResolver(multi: Boolean) on OBJECT

type User @key(fields: "id") @entityResolver(multi: true) {
	id: ID!
	name: String!
}
```

After defining your schema, regenerate the code:

```bash
go run github.com/99designs/gqlgen
```

### Implementation

Implement the generated resolver method:

```go
// IMPORTANT: The output slice order is critical and must match the input slice order exactly!
func (r *entityResolver) FindUserByIDs(ctx context.Context, reps []*entity.UserByIDsInput) ([]*model.User, error) {
	output := make([]*model.User, len(reps))
	for i, user := range reps {
		output[i] = &model.User{
			ID:   user.ID,
			Name: "User " + user.ID,
		}
	}

	return output, nil
}
```

When configured, the federation plugin creates an entity resolver that accepts a list of representations, improving performance by reducing the number of individual resolver calls.

> **Resolver contract.** Return a newly-allocated slice of the same length and
> order as the input — `out[i]` must correspond to `reps[i]` (the runtime places
> results by position). Treat the input representations as read-only; don't
> retain or mutate them.

### Preloaded: @requires data in one scope

By default a multi entity resolver receives only the entity's `@key` fields (in
a generated `…ByKeysInput` struct). Any `@requires` fields are unmarshaled onto
each returned entity *after* the resolver runs, one entity at a time. That is a
problem when the work you do with `@requires` data is naturally batched — for
example, one machine-learning inference call that scores every entity at once —
because the resolver never sees all entities' `@requires` data together.

Enable `federation.options.preloaded_requires` to change the
multi resolver so each input element carries both the `@key` fields **and** the
entity's `@requires` fields, populated *before* the resolver is called:

```yml
federation:
  filename: graph/federation.go
  package: graph
  options:
    preloaded_requires: true
```

With the option set, the input struct gains the `@requires` fields, so the whole
batch's `@requires` data is available in a single call:

```go
// reps[i].Category is a @requires field, populated before this call.
func (r *entityResolver) FindManyProductByIDs(
	ctx context.Context,
	reps []*model.ProductByIDsInput,
) ([]*model.Product, error) {
	// All products' @requires data is visible here at once — score the batch
	// in a single pass instead of once per product.
	return scoreBatch(ctx, reps)
}
```

Notes and limitations:

- Only **flat scalar and enum** `@requires` fields are supported. gqlgen can
  only reconstruct scalar leaves of a representation, so a `@requires` naming an
  object or list field, or a nested path such as `@requires(fields: "world { foo }")`,
  is rejected at generation time. Require the scalar leaves instead
  (`@requires(fields: "world { foo }")` → require `foo`).
- It cannot be combined with `explicit_requires` or `computed_requires`: those
  strategies own `@requires` handling in incompatible ways, so the generator
  rejects the combination rather than silently dropping data.

### Per-entity errors in a batch

A multi entity resolver normally returns `([]*T, error)`; a non-nil error fails
the **whole** batch group. To fail only specific entities while the rest still
resolve, return a `graphql.BatchErrorList` — a slice the same length as the
batch, with a non-nil entry for each entity that failed and `nil` for the ones
that succeeded:

```go
func (r *entityResolver) FindManyProductByIDs(
	ctx context.Context,
	reps []*model.ProductByIDsInput,
) ([]*model.Product, error) {
	out := make([]*model.Product, len(reps))
	errs := make([]error, len(reps))
	var failed bool
	for i, rep := range reps {
		p, err := r.load(ctx, rep)
		if err != nil {
			errs[i] = err // this entity fails…
			failed = true
			continue
		}
		out[i] = p // …this one succeeds
	}
	if failed {
		return out, graphql.BatchErrorList(errs)
	}
	return out, nil
}
```

The generated runtime nulls each failed entity, reports its error against the
`_entities[index]` response path, and still returns the entities that
succeeded. Returning any other (non-`BatchErrors`) error preserves the original
all-or-nothing behavior for the group. This works for every multi entity
resolver, with or without `preloaded_requires`.
