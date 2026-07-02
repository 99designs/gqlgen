# Multi-entity resolution in federation: `@requires` delivery strategies

This directory explores how gqlgen overcomes the N+1 problem in entity resolution, and documents the four strategies for delivering `@requires` data to an entity resolver: `default`, `explicit`, `computed`, and `preloaded`.

The motivating requirement is *single-scope access*: a batch (multi) resolver that sees the `@requires` fields for **all** entities in the batch at once, so a naturally-batched computation can run a single time. The **`preloaded`** strategy delivers exactly this — for scalar/enum `@requires` — by populating the representation the resolver receives *before* it runs. The one case it cannot reach (object-typed/nested `@requires`, such as this README's `variations`) and the strategy that covers that case (`computed`) are documented below.

## Case study

The front-facing API structure has a query, `showProducts`, that is meant to be like a shop front, it returns a list of products.

```graphql
# overall composed schema
type Query {
    showProducts: [Product!]
}
```

The Product subgraph owns the databases, etc related to products.

```graphql
# product subgraph
type Product {
    id: ID!
    variations: [Variation!]
}

type Variation {
    id: ID!
    imageUrl: String!
    price: Float!
}
```

Products can have variations, in that e.g. a wallet might come in blue or black.
There's no guarantee that the variations have the same price either.
But on the front page we should be showing only one variation.

We therefore have a presentation subgraph that internally is connected to some ML model. The ML model
can take a list of variations, and the user ID (from auth token), then choose a particular variation
to recommend to the user. E.g. it might choose blue for people who like the colour blue, black otherwise.

The ML model is expensive to run for single inferences and is better in batch. So our aim is to give it a 2D array
of `[][]Variation`, have it choose a particular variation for each, then return `[]Variation` which would be the
recommended variations for the users.

```graphql
# presentation subgraph
type Product @key(fields: "id") {
    id: ID!
    display: Variation! @requires(fields: "variations {price imageUrl id}")
}

type Variation {
    id: ID!
    imageUrl: String!
    price: Float!
}
```

The website will then make the query to get the variation to render the page:

```graphql
query shoppage {
  showProducts {
    ... on Product {
      id
      display {
        id
        imageUrl
        price
      }
    }
  }
}

```

The examples in the folders above will focus on the presentation subgraph and many different attempts to generate
code using `gqlgen` to achieve the desired outcome.

**The key requirement here is that some part of the function scope must have access to all of the @required fields from all of the
products at once.** This is the constraint we evaluate each strategy against. The `preloaded` strategy satisfies it for scalar/enum `@requires`; the object-typed `variations` field in this exact study is the one shape it cannot reconstruct, which we return to at the end.

The examples will be using the documented [recipes](https://gqlgen.com/v0.17.89/recipes/federation/) for federation
on gqlgen@v0.17.93, the latest version at the time of writing.

## Running the examples

`cd` into the example directory and:

```sh
go run server.go
```

In another terminal, run the entity resolution curl:

```sh
curl --request POST \
  --url http://localhost:8080/query \
  --header 'content-type: application/json' \
  --data '{"query":"query entityresolution(  $representations: [_Any!]!) {  _entities(representations: $representations) {    ... on Product {      id      display {        id      }    }  }}","variables":{  "representations": [    {      "__typename": "Product",      "id": "1",      "variations": [        {          "id": "1a",          "price": 1.00,          "imageUrl": "1a.png"        },        {          "id": "1b",          "price": 2.00,          "imageUrl": "1b.png"        }      ]    },    {      "__typename": "Product",      "id": "2",      "variations": [        {          "id": "2a",          "price": 3.00,          "imageUrl": "2a.png"        },        {          "id": "2b",          "price": 4.00,          "imageUrl": "2b.png"        }      ]    }  ]}}'
```

## Background: Two Entity Resolution Paths

gqlgen generates **two separate code paths** for the `_entities` query:

- **`resolveEntity`** (single): called once per representation — the N+1 path
- **`resolveManyEntities`** (batch): called once per *type*, receiving all N representations at once

`@entityResolver(multi: true)` routes a type to the batch path. Without it, every entity lookup is a separate call.

> **Roadmap.** A future major version plans to make the batch path the default
> (single resolution becomes "multi with N = 1"), reached in two steps: first
> unify the single resolver's signature onto the multi input struct
> (`Find(ctx, rep *…Input)`), then flip the default. Declaring `multi:`
> explicitly today makes that transition a no-op for your entities. See
> `PLAN.md` (Deferred) for the sequencing.

______________________________________________________________________

## The Two Distinct N+1 Problems

Discussions about `@requires` and batching often conflate two different N+1 problems. Separating them is the only way to evaluate each mode honestly.

**N+1 (a) — Entity-fetch N+1.** The subgraph itself makes N separate fetches (database queries, RPC calls, etc.) when resolving N entity representations. This is the classic problem `@entityResolver(multi: true)` addresses.

**N+1 (b) — Per-entity computation N+1.** The subgraph has a unit of work that is naturally batched (such as the ML inference in the case study above, or a single bulk-write to an external system) and needs to see `@requires` data for all N entities in a single function scope. Without that, each per-entity computation becomes an independent call.

The two questions are independent. A mode can solve (a) without solving (b). The case study in this repository specifically requires solving (b) — the ML model must be called once with all N entities' variations.

______________________________________________________________________

## Where `@requires` Data Lives at Each Stage

Understanding what the `@requires` data looks like at each step is essential before evaluating any mode.

**On the wire**: By the time the Apollo Router calls this subgraph's `_entities` query, it has already fetched the required fields from their owning subgraph and embedded them directly in the representation JSON:

```json
{
  "__typename": "MultiHelloRequires",
  "name":  "first name - 1",
  "key1":  "key1 - 1"
}
```

(`name` above is a key field; `key1` is the `@requires` field, pre-fetched by the router.)

So `@requires` fields arrive as data in the request body. The subgraph never needs to issue another network call to fetch them.

**Inside the runtime**: The generated runtime parses the JSON into an `EntityRepresentation` (`map[string]any`) and holds it as `reps[i].entity`. At this point both the key fields and the `@requires` fields are accessible by map lookup.

**In the resolver parameter**: This is the stage each strategy differs on. By default the generated runtime strips each representation down to a key-fields-only input struct before calling the user's resolver. For `@entityResolver(multi: true)`, the resolver parameter is `[]*<Entity>By<Keys>sInput`, where `<Entity>By<Keys>sInput` is a synthetic struct containing only the entity's `@key` fields. Under the default, explicit, and computed strategies the `@requires` data is *not* passed in here. The **`preloaded`** strategy changes exactly this stage: it also unmarshals the scalar `@requires` fields onto that input struct *before* the resolver runs, giving the batch resolver single-scope access to all N entities' `@requires` data (see *The four strategies* below).

**On the returned entity**: Under the default and explicit strategies, after the user's resolver returns, the generated runtime patches `@requires` values onto each `*<Entity>` by reading from the original `reps[i].entity` map. This is in-memory work; no I/O. But by this point the batch resolver has already returned, so there is no longer a function scope in which the user sees all N entities' `@requires` data together — which is precisely why `preloaded` moves the work to the input stage above.

______________________________________________________________________

## The Mechanism: Three Phases in Generated Code

From `federation.gotpl` lines 229–290 and the generated output in `testdata/entityresolver/generated/federation.go` lines 476–509:

**Phase 1 — Build typed input array from key fields only:**

```go
typedReps := make([]*model.MultiHelloRequiresByNamesInput, len(reps))
for i, rep := range reps {
	id0, err := ec.unmarshalNString2string(ctx, rep.entity["name"]) // key field only
	typedReps[i] = &model.MultiHelloRequiresByNamesInput{Name: id0}
}
```

The input struct (`MultiHelloRequiresByNamesInput`) contains *only key fields*. The `@requires` values in `rep.entity` are not copied across. Note that this is a deliberate design choice in the generator — the data is present in `rep.entity` and could in principle be made available to the resolver, but the generated code does not surface it.

**Phase 2 — Single batch call with all N representations:**

```go
entities, err := ec.Resolvers.Entity().FindManyMultiHelloRequiresByNames(ctx, typedReps)
```

One call. All N entities. The resolver handles a single bulk fetch (SQL `WHERE id IN (...)`, Redis pipeline, and so on). This is the resolution to N+1 (a) — entity-fetch batching.

**This is also the only function scope that runs with all N representations active.** It receives only key fields. Any work that needs `@requires` data — including any work that would benefit from seeing it for all N entities at once, such as the ML inference in the case study — cannot run here.

**Phase 3 — Extract `@requires` fields from the original representation:**

```go
for i, entity := range entities {
	entity.Key1, err = ec.unmarshalNString2string(ctx, reps[i].entity["key1"])
	list[reps[i].index] = entity
}
```

Pure in-memory work — no I/O. `reps[i].entity` holds the original representation map that arrived in the request. The loop pulls the pre-fetched values out and assigns them to Go struct fields.

This phase runs *after* the batch resolver returns, so it cannot feed `@requires` data back into the batched computation. Each iteration sees only one entity at a time.

The schema that exercises this (`testdata/entityresolver/schema.graphql`):

```graphql
type MultiHelloRequires @key(fields: "name") @entityResolver(multi: true) {
    name: String! @external
    key1: String! @external
    key2: String! @requires(fields: "key1")
}
```

The generated resolver interface:

```go
FindManyMultiHelloRequiresByNames(ctx context.Context, reps []*model.MultiHelloRequiresByNamesInput) ([]*model.MultiHelloRequires, error)
```

______________________________________________________________________

## The Four Strategies for Delivering `@requires` Fields

gqlgen offers four strategies. Three — `default`, `explicit`, and `preloaded` — describe how `@requires` reaches the entity resolver and are selectable per entity via `@entityResolver(requires: "…")`, with a package-level default in `federation.options`. The fourth, `computed`, routes `@requires` to standalone field resolvers instead, so it is off that axis and is selected only by the `computed_requires` package option. All four solve N+1 (a) when combined with `@entityResolver(multi: true)`. What differs between them is *where* the `@requires` data is surfaced to your code — and, decisively for the case study, whether the batch resolver can see all `@requires` data at once. Only **`preloaded`** surfaces it *before* the resolver runs, so only `preloaded` solves N+1 (b) (for scalar/enum `@requires`); the others deliver it one entity at a time — `default`/`explicit` after the resolver returns, `computed` to a per-field resolver.

### 1. Default (implicit) — direct unmarshal from representation

The generated code does the work. In Phase 3 above, the template emits:

```go
entity.{{.Field.JoinGo `.`}}, err = ec.{{.Type.UnmarshalFunc}}(ctx, reps[i].entity["{{...}}"])
```

For nested `@requires` (for example, `@requires(fields: "world{ foo }")`), it chains map access:

```go
entity.World.Foo, err = ec.unmarshalNString2string(ctx, reps[i].entity["world"].(map[string]any)["foo"])
```

The `MultiPlanetRequiresNested` case demonstrates this (generated `federation.go` line 577).

**N+1 (a)**: solved — single batch call.
**N+1 (b)**: not solved — the resolver sees only key fields, and `@requires` is patched onto entities one at a time *after* the resolver returns.

### 2. `explicit_requires: true` — user-implemented population function

When `federation.options.explicit_requires: true` is set, gqlgen generates a stub:

```go
func (ec *executionContext) PopulateMultiHelloRequiresRequires(
	ctx context.Context,
	entity *model.MultiHelloRequires,
	rep map[string]any,
) error
```

The template calls this in Phase 3 instead of the auto-generated field assignment (`federation.gotpl` lines 267–271):

```go
{{- if and $.PackageOptions.ExplicitRequires (index $.RequiresEntities $entity.Def.Name) }}
    err = ec.Populate{{$entity.Def.Name}}Requires(ctx, entity, reps[i].entity)
```

**N+1 (a)**: solved — single batch call.
**N+1 (b)**: not solved — `PopulateMultiHelloRequiresRequires` is called per entity inside the post-resolver loop. Useful when `@requires` fields need transformation or validation before assignment, but it does not provide a scope where all N entities' `@requires` data is visible at once.

See `entityresolverexplicit/` in this repository for the resulting resolver shape.

### 3. `computed_requires: true` — standard field resolvers (Federation v2 only)

With `computed_requires`, the template emits nothing in Phase 3:

```go
{{- if $options.ComputedRequires }}
    {{/* We don't do anything in this case, computed requires are handled by standard resolvers */}}
```

Instead, gqlgen treats `@requires` fields as ordinary field resolvers that receive the representation value via a directive argument. The batch entity-fetch still runs once.

**N+1 (a)**: solved at the entity-fetch level.
**N+1 (b)**: not solved — the `@requires` data is delivered to a *field* resolver (e.g., `Display(ctx, obj *Product, federationRequires map[string]any)`), which is called once per entity in the result. There is no scope in which all N entities' `@requires` data is visible together. See `entityresolvermulti/` in this repository for an explicit demonstration of this limitation.

Also requires `call_argument_directives_with_null: true` in `gqlgen.yml`, and only works with Federation version 2.

### 4. `preloaded_requires: true` — populate the resolver's input representation

This is the strategy built for N+1 (b). With `preloaded_requires` (or `@entityResolver(requires: "preloaded")` on a single entity), gqlgen unmarshals the scalar `@requires` fields onto the multi resolver's input struct in Phase 1 — *before* the batch call — instead of patching them onto the returned entity in Phase 3:

```go
// key1 is a @requires field, populated onto the input before the resolver runs.
typedReps[i] = &model.MultiHelloRequiresByNamesInput{Name: id0, Key1: key1}
```

The resolver signature is unchanged (`FindMany<Entity>ByNames(ctx, reps []*<Entity>By<Keys>sInput)`); the `@requires` fields simply arrive already populated on each element of `reps`:

```go
func (r *entityResolver) FindManyMultiHelloRequiresByNames(
    ctx context.Context,
    reps []*model.MultiHelloRequiresByNamesInput,
) ([]*model.MultiHelloRequires, error) {
    // Every reps[i].Key1 is available here — one scope, all N entities.
    return scoreBatch(ctx, reps)
}
```

**N+1 (a)**: solved — single batch call.
**N+1 (b)**: **solved** — every entity's `@requires` data is visible in one function scope, so a naturally-batched computation runs once. This is the property no other strategy provides.

The `@requires` fields are added to the generated input type as modelgen `ExtraFields` (the federation plugin runs before modelgen in the config-mutation phase), and the template populates them from `reps[i].entity` before calling the resolver. The input representation is read-only; the resolver returns a freshly-allocated `[]*<Entity>` of the same length and order. See `entityresolverpreloaded/` in this repository for the resulting resolver shape.

**Requires `multi: true`, and scalar/enum `@requires` only.** gqlgen can reconstruct only scalar leaves of a representation — output object types have no unmarshaler — so a `preloaded` `@requires` naming an object or list field, or a nested path such as `@requires(fields: "world { foo }")`, is rejected at generation time. Mark such a field `@goComputed` to route just that field to a standalone field resolver (delivered via `federationRequires`) while the entity's scalar `@requires` stay preloaded on the input — both on the same entity.

______________________________________________________________________

## How `preloaded` Satisfies the Case Study — and the One Shape It Cannot

The case study at the top of this README has this requirement, restated for emphasis:

> Some part of the function scope must have access to all of the @required fields from all of the products at once.

Concretely, the presentation subgraph needs a function scope that sees every product's `@requires` data at once so it can make a single ML inference call across the whole batch.

Tracing the case study through each strategy:

| Strategy                     | What the batch resolver receives                            | Where `@requires` data is available                                              | Single-scope access to all `@requires`?       |
| ---------------------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------------------------------- |
| Default                      | `[]*ProductByIDsInput` (IDs only)                           | Patched onto each `*Product` *after* the batch resolver returns                   | No                                            |
| `explicit_requires: true`    | `[]*ProductByIDsInput` (IDs only)                           | Visible inside `PopulateProductRequires(ctx, entity, rep)`, called once per entity *after* the batch resolver returns | No                                            |
| `computed_requires: true`    | `[]*ProductByIDsInput` (IDs only)                           | Passed to the field resolver `Display(ctx, obj, federationRequires)`, called once per entity at field-resolution time | No                                            |
| `preloaded_requires: true`   | `[]*ProductByIDsInput` **with `@requires` fields populated** | On each `reps[i]` *before* the resolver runs — all N entities in one scope        | **Yes** (scalar/enum `@requires`)             |

`preloaded` is the strategy that meets the requirement: it hands the batch resolver every entity's scalar `@requires` data in a single function scope, so the batched computation runs once. The data exists in `reps[i].entity` inside the generated runtime during Phase 1, and `preloaded` surfaces it there. This was always possible within the federation protocol — nothing in the wire contract prevents the resolver from receiving the pre-fetched representation — and `preloaded` is where the generator does so.

**The one shape `preloaded` alone cannot reach.** This README's case study asks for `[][]Variation` — an *object-typed* `@requires` (`variations { price imageUrl id }`). gqlgen can reconstruct only scalar leaves of a representation; output object types have no unmarshaler, so `preloaded` rejects object/list/nested `@requires` at generation time. Mark that field `@goComputed` and it is delivered to a standalone field resolver via `federationRequires`, while the entity's *scalar* `@requires` stay preloaded on the batch input — so one entity gets single-scope batching for its scalars and per-field delivery for its object field. (Alternatively, require only the scalar leaves you actually need so `preloaded` can carry them.) The remaining gap is single-scope batching *of the object data itself*, which needs gqlgen to gain representation-level unmarshalers for composite types.

The examples in this repository each demonstrate one strategy against the case study: `entityresolverpreloaded/` (the `preloaded` strategy that meets the requirement), and `entityresolvermulti/`, `entityresolverexplicit/`, `batchfieldcomputed/`, `batchfieldexplicit/` (the other strategies and where each surfaces `@requires`).

______________________________________________________________________

## The `entity_resolver_multi` Package-Level Option

Rather than annotating every type with `@entityResolver(multi: true)` in the schema, you can set:

```yaml
federation:
  options:
    entity_resolver_multi: true
```

This makes multi the default for all entity resolvers (`PackageOptions.EntityResolverMulti`). Per-type `@entityResolver(multi: false)` can override it back to single.

______________________________________________________________________

## Ordering Guarantee

The batch resolver contract requires that the output slice is the **same length and same order** as the input slice. The generated code maintains a parallel index (`EntityWithIndex{index: i, entity: rep}`) so that when results come back, each entity is placed at the correct position in the final result list via `list[reps[i].index] = entity`. Your batch resolver implementation must respect this ordering or results will be misassigned across clients.

______________________________________________________________________

## Summary

The summary below intentionally distinguishes the two N+1 questions so that "no N+1" claims do not paper over the case study's actual requirement.

| Scenario                                                           | Entity-fetch N+1 (a)?                                | `@requires` accessible inside the batch resolver scope (b)?                                                                                                              |
| ------------------------------------------------------------------ | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| No `@entityResolver`                                               | Yes                                                  | N/A — `resolveEntity` is called once per representation, so per-entity computation is the default anyway                                                                  |
| `@entityResolver(multi: true)`, no `@requires`                     | No — single batch call                               | N/A — no `@requires` fields exist                                                                                                                                         |
| `@entityResolver(multi: true)` + `@requires` (default mode)        | No — single batch call                               | **No** — resolver receives `[]*<Entity>By<Keys>sInput` (keys only); `@requires` patched onto entities one at a time *after* the resolver returns                          |
| `@entityResolver(multi: true)` + `@requires` + `explicit_requires` | No — single batch call                               | **No** — `Populate<Entity>Requires(ctx, &entity, rep)` runs once per entity inside the post-resolver loop                                                                 |
| `@entityResolver(multi: true)` + `@requires` + `computed_requires` | No, for entity fetch — single batch call             | **No** — `@requires` is delivered to per-field resolvers via a `federationRequires map[string]any` argument, called once per entity at field-resolution time              |
| `@entityResolver(multi: true)` + `@requires` + `preloaded_requires` | No — single batch call                               | **Yes** — scalar/enum `@requires` populated onto each `reps[i]` *before* the resolver runs, so all N entities are visible in one scope (object/nested `@requires` on the same entity go via `@goComputed`) |

The right way to read the second column: the `@requires` data is always *somewhere* — it lives in `reps[i].entity` inside the generated runtime, and ends up either on the returned entity, in a field resolver's arguments, or (with `preloaded`) on the resolver's input representation. `preloaded` is the strategy that hands the batch resolver direct access to scalar `@requires` values for all N entities in the same function scope, resolving N+1 (b) without changing the resolver signature.

Object-typed/nested `@requires` are handled per field by `@goComputed`, which routes just that field to a standalone field resolver while the entity's scalar `@requires` stay preloaded; the remaining gap is single-scope batching *of the object data itself*, which needs gqlgen to gain representation-level unmarshalers for composite types. This repository documents each strategy against a concrete case study so the trade-offs are visible side by side.
