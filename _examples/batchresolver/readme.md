# Batch Resolver Example

This example demonstrates **batch field resolvers** in gqlgen — resolvers that receive a slice of parent objects and return results for all of them in a single call, instead of being invoked once per parent.

## Schema

A `User` type has six `Profile` fields covering the key variations:

| Field                      | Nullable | Batch | Has Args |
|----------------------------|----------|-------|----------|
| `nullableBatch`            | yes      | yes   | no       |
| `nullableNonBatch`         | yes      | no    | no       |
| `nullableBatchWithArg`     | yes      | yes   | yes      |
| `nullableNonBatchWithArg`  | yes      | no    | yes      |
| `nonNullableBatch`         | no       | yes   | no       |
| `nonNullableNonBatch`      | no       | no    | no       |

## Configuration

In `gqlgen.yml`, batch resolvers are enabled per-field:

```yaml
models:
  User:
    fields:
      nullableBatch:
        resolver: true
        batch: true
```

This changes the generated resolver signature from the standard single-object form:

```go
NullableNonBatch(ctx context.Context, obj *User) (*Profile, error)
```

to a batch form that receives all parents at once:

```go
NullableBatch(ctx context.Context, objs []*User) ([]*Profile, error)
```

## Per-Item Errors

Batch resolvers can return per-item errors using `graphql.BatchErrorList`:

```go
results := make([]*Profile, len(objs))
errs := make([]error, len(objs))
for i, obj := range objs {
    results[i], errs[i] = resolve(obj)
}
return results, graphql.BatchErrorList(errs)
```

Each entry in the error slice corresponds to the parent at the same index. Individual errors can also be `gqlerror.List` to report multiple errors for a single item.

## Nested Batching

The schema also demonstrates **nested batch resolvers** through the path `User → Profile → Image`:

| Parent    | Field             | Batch | Target  |
|-----------|-------------------|-------|---------|
| `User`    | `profileBatch`    | yes   | Profile |
| `User`    | `profileNonBatch` | no    | Profile |
| `Profile` | `coverBatch`      | yes   | Image   |
| `Profile` | `coverNonBatch`   | no    | Image   |

With 10 users, the batch path resolves all profiles in **1 call** (vs 10 for non-batch). However, `coverBatch` is still called **once per profile** (10 calls) rather than once for all profiles. This happens because profiles returned by a batch resolver are marshalled as individual values, not as a list — the batch parent context for `Profile` is only set when marshalling a `[Profile]` list field. Ideally, nested batching should propagate the batch parent context from batch resolver results so that `coverBatch` is called only once for all 10 profiles. The `TestBatchResolver_Nested_CallCount` test documents these current call counts and confirms both paths return identical data.

## Tests

The tests verify **parity** between batch and non-batch resolvers — both must produce identical data and errors for the same inputs. Covered scenarios include:

- Successful resolution
- Arguments passed through correctly
- Errors at specific indices
- `gqlerror.List` expansion
- `gqlerror.Error` with and without a custom path
- Non-null field error propagation (parent nulled out)
- Wrong result/error slice lengths (produces per-parent error messages)
- Nested batch call count verification (`User → Profile → Image`)

## Benchmarks

`BenchmarkBatchResolver_SingleLevel` and `BenchmarkBatchResolver_Nested` compare batch vs non-batch execution time. Note that these benchmarks use in-memory resolvers with no I/O, so they only measure the framework overhead of batching. In a real-world scenario the main benefit of batching is reducing the number of round-trips to external services (databases, APIs, etc.), which these benchmarks do not capture.
