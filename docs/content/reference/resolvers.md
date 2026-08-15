---
linkTitle: Resolvers
title: Resolving graphQL requests
description: Different ways of binding graphQL requests to resolvers
menu: { main: { parent: 'reference', weight: 10 } }
---

There are multiple ways that a graphQL type can be bound to a Go struct that allows for many usecases.


## Bind directly to struct field names
This is the most common use case where the names of the fields on the Go struct match the names of the
fields in the graphQL type.  If a Go struct field is unexported, it will not be bound to the graphQL type.

```go
type Car struct {
    Make string
    Model string
    Color string
    OdometerReading int
}
```

And then in your graphQL schema:
```graphql
type Car {
    make: String!
    model: String!
    color: String!
    odometerReading: Int!
}
```

And in the gqlgen config file:
```yaml
models:
    Car:
        model: github.com/my/app/models.Car
```

In this case, each field in the graphQL type will be bound to the respective field on the go struct
ignoring the case of the fields


## Bind to a method name

This is also very common use case that comes up where we want to bind a graphQL field to a Go struct method

```go
type Person struct {
    Name string
}

type Car struct {
    Make string
    Model string
    Color string
    OwnerID *string
    OdometerReading int
}

func (c *Car) Owner() (*Person) {
    // get the car owner
    //....
    return owner
}
```

And then in your graphQL schema:
```graphql
type Car {
    make: String!
    model: String!
    color: String!
    odometerReading: Int!
    owner: Person
}
```

And in the gqlgen config file:
```yaml
models:
    Car:
        model: github.com/my/app/models.Car
    Person:
        model: github.com/my/app/models.Person
```

Here, we see that there is a method on car with the name ```Owner```, thus the ```Owner``` function will be called if
a graphQL request includes that field to be resolved.

Model methods can optionally take a context as their first argument. If a
context is required, the model method will also be run in parallel.

## Bind when the field names do not match

There are two ways you can bind to fields when the Go struct and the graphQL type do not match.


The first way is you can bind resolvers to a struct based off of struct tags like the following:

```go
type Car struct {
    Make string
    ShortState string
    LongState string `gqlgen:"state"`
    Model string
    Color string
    OdometerReading int
}
```

And then in your graphQL schema:
```graphql
type Car {
    make: String!
    model: String!
    state: String!
    color: String!
    odometerReading: Int!
}
```

And in the gqlgen config file add the line:
```yaml
struct_tag: gqlgen

models:
    Car:
        model: github.com/my/app/models.Car
```

Here even though the graphQL type and Go struct have different field names, there is a Go struct tag field on ```longState```
that matches and thus ```state``` will be bound to ```LongState```.


The second way you can bind fields is by adding a line into the config file such as:
```go
type Car struct {
    Make string
    ShortState string
    LongState string
    Model string
    Color string
    OdometerReading int
}
```

And then in your graphQL schema:
```graphql
type Car {
    make: String!
    model: String!
    state: String!
    color: String!
    odometerReading: Int!
}
```

And in the gqlgen config file add the line:
```yaml
models:
    Car:
        model: github.com/my/app/models.Car
        fields:
            state:
                fieldName: LongState
```

## Binding to Anonymous or Embedded Structs
All of the rules from above apply to a struct that has an embedded struct.
Here is an example
```go
type Truck struct {
    Car

    Is4x4 bool
}

type Car struct {
    Make string
    ShortState string
    LongState string
    Model string
    Color string
    OdometerReading int
}
```

And then in your graphQL schema:
```graphql
type Truck {
    make: String!
    model: String!
    state: String!
    color: String!
    odometerReading: Int!
    is4x4: Bool!
}
```

Here all the fields from the Go struct Car will still be bound to the respective fields in the graphQL schema that match

Embedded structs are a good way to create thin wrappers around data access types an example would be:

```go
type Cat struct {
    db.Cat
    //...
}

func (c *Cat) ID() string {
    // return a custom id based on the db shard and the cat's id
     return fmt.Sprintf("%d:%d", c.Shard, c.Id)
}
```

Which would correlate with a gqlgen config file of:
```yaml
models:
    Cat:
        model: github.com/my/app/models.Cat
```

## Binding Priority
If a ```struct_tags``` config exists, then struct tag binding has the highest priority over all other types of binding.
In all other cases, the first Go struct field found that matches the graphQL type field will be the field that is bound.

## Batch resolvers

A field marked with `@goField(batch: true)` — or every resolver field, when
`resolver.batch` is enabled — receives all of its sibling parent objects in
one call instead of being called once per parent:

```go
// Instead of:
//   Node(ctx context.Context, obj *ProfileEdge) (*Profile, error)
func (r *profileEdgeResolver) Node(
	ctx context.Context, objs []*ProfileEdge,
) ([]*Profile, error)
```

Return one result per parent, in the same order as `objs`. gqlgen matches
results to parents by index.

Batch resolvers are only available on types that gqlgen does not resolve
itself. Root types, input objects, introspection types (`__*`) and the
federation `_Service`/`Entity` types are excluded; asking for a batch resolver
on one is a code generation error that names the reason.

### Do not retain or mutate the parents

`objs` always arrives as `[]*T`, but where those pointers come from depends on
how the parent list is modelled in Go:

- For a `[]*T` field, they are the pointers your own resolver returned.
- For a `[]T` field — which is what
  [`omit_slice_element_pointers`](/config) produces, and what an explicit
  `type: "[]pkg.T"` mapping produces — gqlgen takes the address of each slice
  element. Those pointers alias the caller's backing array.

In the second case the slice is being marshaled concurrently while your
resolver runs, so:

- **Do not mutate `objs[i]`.** Writing through the pointer races with the
  marshaler reading the same element, and the write is visible to whatever
  produced the slice.
- **Do not retain `objs` beyond the call.** The backing array belongs to the
  parent resolver, not to you.
- **Do not use pointer identity to correlate parents.** For a value slice, the
  pointer handed to a batch resolver is not the same pointer the equivalent
  non-batch resolver would receive — the non-batch path takes the address of a
  copy. Correlate by index, or by a key on the parent.

Treat `objs` as read-only input and return a freshly allocated result slice.
