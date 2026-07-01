# Batch field resolver + explicit requires

This is the example that works; pairing the gqlgen.yml parameters `explicit_requires: true`, `entity_resolver_multi: true` with
`@goField(batch: true)` directive on the field definition gives us a function scope with access
to all of the required fields provided they are unmarshalled in `federation.requires.go`.

However, this only works for one field at a time. What if there's two fields that need to be resolved from the same database,
with the requirement to use 1 database call for both fields?

Ideally there should be some way of generating a function signature of `[]*Product -> []*Product` where the required fields are
already populated inside the input arguments. If there's a possible way of doing this in gqlgen currently please do share.
