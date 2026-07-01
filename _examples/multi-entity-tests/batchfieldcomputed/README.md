# Batch fields + computed

This example attempts to use `entity_resolver_multi: true` with `computed_requires: true` and `@goField(batch: true)` declared in the schema.

There is a batch Display resolver generated, however the `federatedRequires` obj passed into it contains only one @requires object
when there should be an array of required objects passed into it.

This is evident when observing the stdout output when running the example query, which is printed at `schema.resolvers.go:19`.
