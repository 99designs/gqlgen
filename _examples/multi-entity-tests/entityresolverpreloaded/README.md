# Entity resolver multi + preloaded requires

This approach uses the following gqlgen config:

```yml
# Uncomment to enable federation
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
    preloaded_requires: true
    entity_resolver_multi: true
```

Unlike the [`entityresolvermulti`](../entityresolvermulti) example — where the
`@requires` data only ever reaches a *per-entity field resolver*, one product at
a time — `preloaded_requires` populates each entity's `@requires` fields onto the
batch resolver's **input** before it runs. The batch resolver therefore has a
single function scope containing every product's `@requires` data:

```go
// FindManyProductByIDs is the resolver for the findManyProductByIDs field.
func (r *entityResolver) FindManyProductByIDs(ctx context.Context, reps []*model.ProductByIDsInput) ([]*model.Product, error) {
	// The whole batch is visible at once: collect every product's category.
	categories := make([]string, len(reps))
	for i, rep := range reps {
		categories[i] = rep.Category // <-- @requires field, populated before the call
	}
	batch := strings.Join(categories, ",")

	products := make([]*model.Product, len(reps))
	for i, rep := range reps {
		products[i] = &model.Product{
			ID:      rep.ID,
			Display: fmt.Sprintf("%s display (batch of %d: %s)", rep.Category, len(reps), batch),
		}
	}
	return products, nil
}
```

This is the scope needed to eliminate N+1 (b): a naturally-batched computation
(one ML-inference call scoring every product at once, a single bulk write, and so
on) runs a single time over the whole batch instead of once per entity.

There is no `Display` field resolver here — `display` is produced by the batch
resolver from the preloaded `category`, so the entire `@requires` handling lives
in one place.

## Limitation — flat scalar/enum `@requires`

`preloaded_requires` can only reconstruct scalar leaves of a representation;
output object types have no unmarshaler. The parent
[`README.md`](../README.md) case study uses
`@requires(fields: "variations { price imageUrl id }")` (object leaves), so this
example recasts it with a scalar `category` requirement. An entity that needs an
object-typed `@requires` uses the `computed` strategy instead (see
[`entityresolvermulti`](../entityresolvermulti)).

## Running the example

```sh
go run server.go
```

In another terminal, resolve two products in one batch — both representations
already carry their `category` (`@requires`), pre-fetched by the router:

```sh
curl --request POST \
  --url http://localhost:8080/query \
  --header 'content-type: application/json' \
  --data '{"query":"query ($representations:[_Any!]!){ _entities(representations:$representations){ ... on Product { id display } } }","variables":{"representations":[{"__typename":"Product","id":"1","category":"books"},{"__typename":"Product","id":"2","category":"toys"}]}}'
```

Each product's `display` names the whole batch (`books,toys`), demonstrating that
both products' `@requires` data was visible in a single resolver call.
