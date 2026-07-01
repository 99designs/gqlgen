# Federation entity resolvers: `@requires` strategies, batched resolution, and key handling

This is the design for gqlgen's federation entity-resolver `@requires` handling
and batch (`@entityResolver(multi: true)`) resolution. The companion `README.md`
states the use case the design is measured against; the worked examples live in
the sibling directories.

## Summary

- **Four `@requires` strategies**: `default`, `explicit`, `preloaded`, and
  `computed`. The first three describe how `@requires` reaches the entity
  resolver and are selectable **per entity** via `@entityResolver(requires: "…")`,
  with package options as the default. `computed` is the outlier — it routes
  `@requires` to standalone field resolvers, so it is selected only by the
  `computed_requires` package option, not the directive (a per-field
  `@goComputed` is the planned home for it).
- **`preloaded`** — a strategy that hands a batch resolver every
  entity's `@requires` data in a single scope, so a naturally-batched
  computation (e.g. one ML-inference call across the whole batch) can run once.
- **Per-index errors** for batch resolvers via `graphql.BatchErrorList`: one
  entity can fail without sinking the rest of the batch.
- **Key-field collision disambiguation** in generated multi-resolver input
  types.

Everything is backward compatible: with no new directive arguments or options,
existing schemas regenerate byte-for-byte unchanged.

______________________________________________________________________

## Background: two N+1 problems

Discussions of `@requires` and batching conflate two independent problems.

- **N+1 (a) — entity-fetch.** The subgraph makes N separate backend fetches when
  resolving N representations. `@entityResolver(multi: true)` addresses this: one
  `FindMany<X>` call, one `WHERE id IN (...)`.
- **N+1 (b) — per-entity computation.** The subgraph has a unit of work that is
  *naturally batched* — the README's ML model that scores every product at once,
  or a single bulk write — and needs the `@requires` data for **all N entities
  in one function scope** to run once.

The two are orthogonal. Multi mode solves (a). The `preloaded` strategy
solves (b), for the cases gqlgen can represent (see *Strategies* below).

______________________________________________________________________

## How entity resolution works

`_entities(representations: [_Any!]!)` is the only entity entry point the Apollo
Router sees. The generated runtime (`federation.gotpl`) groups representations by
`__typename` and, per group, calls `resolveEntityGroup`, which forks:

- **Non-multi:** one goroutine per representation, each calling `resolveEntity`
  with scalar key arguments.
- **Multi:** a single `resolveManyEntities` call that invokes the user's
  `FindMany<X>` resolver with the whole group.

For multi mode the plugin generates a synthetic input type
(`<Entity>By<Keys>sInput`) that carries the resolver's key fields. This type is
**internal scaffolding**: it is emitted into a `BuiltIn` source that
`__resolve__service` excludes from the SDL served to the router, so its Go shape
can be changed freely without any wire-contract impact.

`@requires` fields arrive in the representation JSON (the router pre-fetches them
from their owning subgraph). The question every strategy answers is *where that
data is surfaced to your code.*

______________________________________________________________________

## The `@requires` strategies

| Strategy  | `requires:` value         | Where `@requires` data is delivered                                                                 |
| --------- | ------------------------- | --------------------------------------------------------------------------------------------------- |
| Default   | `"default"`               | Unmarshaled onto the returned entity, after the resolver runs.                                       |
| Explicit  | `"explicit"`              | A user-implemented `Populate<Entity>Requires` on the returned entity, after the resolver. Supports nested/array fields. |
| Preloaded | `"preloaded"`             | Unmarshaled onto the resolver's **input** representation, before the resolver runs.                  |
| Computed  | _(package option only)_   | Delivered to standalone field resolvers via a `federationRequires` argument (Federation 2 only).     |

The four are mutually exclusive: each entity resolves to exactly one. The first
three share one design decision (how `@requires` reaches the entity resolver), so
combining two would mean two mechanisms fighting over the same fields — the
generator rejects incompatible combinations rather than emit something that
silently drops data. `computed` is off that axis (it routes fields to standalone
field resolvers), which is why it is selected by the `computed_requires` package
option rather than the directive; per-field selection is deferred to a
`@goComputed` field directive (see *Deferred*).

### `preloaded`

This is the strategy for N+1 (b). The multi resolver receives its input with both
`@key` and `@requires` fields populated *before* it runs, so the whole batch's
`@requires` data is visible in one scope:

```go
// reps[i].Category is a @requires field, populated before this call.
func (r *entityResolver) FindManyProductByIDs(
    ctx context.Context,
    reps []*model.ProductByIDsInput,
) ([]*model.Product, error) {
    return scoreBatch(ctx, reps) // one batched pass over every product
}
```

Mechanics: the `@requires` fields are added to the generated input type as
modelgen `ExtraFields` (the federation plugin runs before modelgen in the
config-mutation phase, so no forward reference to a not-yet-generated type is
needed), and the template populates them before calling the resolver. The
resolver returns a newly-allocated `[]*<Entity>` of the same length and order as
the input; the input representations are read-only. Input (`…ByKeysInput`, the
read model) and output (`<Entity>`, the write model) stay distinct types.

**Limitation — flat scalar/enum `@requires` only.** gqlgen can only reconstruct
scalar leaves of a representation; output object types have no unmarshaler. So a
`@requires` naming an object or list field, or a nested path such as
`@requires(fields: "world { foo }")`, is rejected at generation time. Entities
that need object-typed `@requires` use the `computed` strategy instead (see
*Per-entity selection*).

______________________________________________________________________

## Selecting a strategy per entity

A package option sets the default strategy for the whole subgraph; a per-entity
`@entityResolver(requires: "…")` argument overrides it, choosing among `default`,
`explicit`, and `preloaded`. This mirrors how `multi` already resolves per entity
(`isMultiEntity` / `resolveRequiresStrategy`). To mix `computed` (object-typed
`@requires`) with `preloaded` (scalar `@requires`) in one subgraph, make
`computed` the package default and override the scalar entity:

```graphql
directive @entityResolver(multi: Boolean, requires: String) on OBJECT

# with computed_requires: true, computed is the package default.
type Planet  @key(fields: "name")                                                     { ... }
type Product @key(fields: "id")   @entityResolver(multi: true, requires: "preloaded") { ... }
```

Because the choice is a single value per entity, the strategies are mutually
exclusive by construction — there is no combination to validate away at the
entity level. `@entityResolver` is your own directive, so the `requires: String`
argument is added to its definition; a string (rather than an enum) keeps the
same ergonomics as `multi: Boolean`. With no argument, an entity uses the package
default. `computed` is not a `requires:` value; a `requires: "computed"` is
rejected with a message pointing at the `computed_requires` package option.

**Validation** (fail fast, with actionable messages):

- An unknown strategy value names the entity, the bad value, and the valid set;
  the specific value `"computed"` is rejected with a hint to use the
  `computed_requires` package option instead.
- `preloaded` requires `multi: true`.
- `computed` requires Federation 2 and `call_argument_directives_with_null`,
  checked whenever any entity resolves to `computed`.

______________________________________________________________________

## Batched resolution and per-index errors

A multi resolver returns `([]*T, error)`. A plain non-nil error fails the whole
`__typename` group. To fail individual entities while the rest resolve, return a
`graphql.BatchErrorList` — a slice the same length as the batch, with a non-nil
entry for each failed entity:

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
            errs[i], failed = err, true
            continue
        }
        out[i] = p
    }
    if failed {
        return out, graphql.BatchErrorList(errs)
    }
    return out, nil
}
```

The generated runtime nulls each failed entity, reports its error against the
`_entities[index]` path, and returns the entities that succeeded. The split
between per-index errors and a fatal error is `fedruntime.SplitEntityBatchErrors`.
This reuses gqlgen's existing `BatchErrors` mechanism, so the resolver signature
is unchanged and it applies to every multi resolver.

______________________________________________________________________

## Key-field collision disambiguation

A multi resolver's synthetic input type names each field from
`KeyField.Field.ToGo()`. When two key paths in one `@key` reduce to the same Go
name — e.g. `@key(fields: "id i { d }")`, where both `id` and `i { d }` yield
`ID` — the input type would emit a duplicate field and fail schema validation.

Each key field is assigned a name unique within its resolver (`ID`, `ID2`, …),
computed once on `KeyField.GoName` and read by both the SDL builder and the
template, so the SDL field, the modelgen struct field, and the template's struct
literal stay in agreement (the suffixed names are idempotent under `ToGo`). This
affects only multi mode; the non-multi path uses positional parameters. It is a
no-op when there is no collision.

______________________________________________________________________

## Design rationale (Go guidelines)

| Decision                                                            | Rationale                                                                                   |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `preloaded` input is the representation read-model, returns `[]*Entity` | Separate read/write models; no mutation of the input slice.                                  |
| One `RequiresStrategy` value per entity                             | Mutually-exclusive states are unrepresentable; no combinatorial validation at the entity level. |
| `requires:` on `@entityResolver`, resolving like `multi`           | Reuse the existing per-entity surface; no new option axis; a reader who knows `multi` knows `requires`. |
| `computed` selected by the package option, not the `requires:` directive | It routes fields off the entity resolver, so it is off the axis the directive names. Keeps that axis coherent (three members) and leaves no unreleased directive value to deprecate when per-field `@goComputed` lands. |
| Per-index errors via existing `BatchErrors`                        | No resolver-signature change, no core-codegen change; reuse gqlgen's own idiom.              |
| Disambiguate key names on one stored `GoName`                       | One source of truth instead of the same naming decision re-derived in three places.          |
| Reject incompatible strategy combinations at generation time        | Fail fast on contradictory config rather than generate code that silently drops data.        |

______________________________________________________________________

## Deferred

- **Multi as the default**, in two sequenced major-version steps:
  1. **Unify the input type** — change the single-entity resolver signature from
     positional key arguments (`Find(ctx, id string)`) to the same input struct
     the multi resolver already uses (`Find(ctx, rep *<Entity>By<Keys>sInput)`).
     Default stays single; runtime behavior is unchanged (still one goroutine per
     representation), so it is a pure signature change. This also lets `preloaded`
     and key-collision handling apply to single entities.
  2. **Flip the default** to `multi: true` ("multi with N = 1"). `multi: false`
     remains the opt-out and, after step 1, already carries the unified
     signature, so opting out needs no code change. Per-index errors (above)
     remove the historical objection that multi was all-or-nothing. Document the
     tail-latency nuance — the single path resolves representations in parallel,
     so a sequential `FindMany` should fan out internally when its backend cannot
     batch.

  Step 1 goes first so the default flip is a pure arity change on a shared input
  type, letting both single- and batch-preferring users reach their final
  signature in one migration. Neither step affects non-federation users.
- **Object/nested `@requires` for `preloaded`**, which needs gqlgen to
  gain representation-level unmarshalers for composite types. Until then,
  `computed` covers those entities.
- **Per-field `computed` via a `@goComputed` field directive**, replacing the
  entity-level all-or-nothing `computed_requires`. It would let one entity
  compute an object-typed `@requires` field while preloading its scalar
  `@requires` fields onto the batch input, and make `@goComputed` the single
  directive-surface way to select computed.
