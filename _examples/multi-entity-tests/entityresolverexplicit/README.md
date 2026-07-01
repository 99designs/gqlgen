# Entity resolver multi + explicit requires

This approach uses the following gqlgen config:

```yml
# Uncomment to enable federation
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
   explicit_requires: true
   entity_resolver_multi: true
```

This approach doesn't work because of the following order of events in `federation.go:201`:

```go
			entities, err := ec.Resolvers.Entity().FindManyProductByIDs(ctx, typedReps)
			if err != nil {
				return err
			}

			for i, entity := range entities {
				err = ec.PopulateProductRequires(ctx, entity, reps[i].entity)
				if err != nil {
					return fmt.Errorf(`populating requires for Entity "Product": %w`, err)
				}

				list[reps[i].index] = entity
			}
			return nil
```

The multi entity resolver has access to only the entity key fields, it then returns and is never accessed again. Then the @requires
fields are populated by the explicit requires method in `federation.requires.go`. There's no point at which we have access to the full list
of products with variations populated.

This is evident in the order of logs in stdout when running the example query, which are generated from the println lines at `entity.resolver.go:18` and `federation.requires.go:13`:

```
[
  {
    "ID": "1"
  },
  {
    "ID": "2"
  }
]
{"__typename":"Product","id":"1","variations":[{"id":"1a","imageUrl":"1a.png","price":1.00},{"id":"1b","imageUrl":"1b.png","price":2.00}]}
{"__typename":"Product","id":"2","variations":[{"id":"2a","imageUrl":"2a.png","price":3.00},{"id":"2b","imageUrl":"2b.png","price":4.00}]}
```


