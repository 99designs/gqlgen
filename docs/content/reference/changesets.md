---
linkTitle: Changesets
title: Using maps as changesets
description: Falling back to map[string]interface{} to allow for presence checks.
menu: { main: { parent: 'reference', weight: 10 } }
---

Occasionally you need to distinguish presence from nil (undefined vs null). In gqlgen this can be done using either maps or the Omittable type.

## Maps

```graphql
type Mutation {
	updateUser(id: ID!, changes: UserChanges!): User
}

input UserChanges {
	name: String
	email: String
}
```

Then in config set the backing type to `map[string]interface{}`
```yaml
models:
  UserChanges:
    model: "map[string]interface{}"
```

After running go generate you should end up with a resolver that looks like this:
```go
func (r *mutationResolver) UpdateUser(ctx context.Context, id int, changes map[string]interface{}) (*User, error) {
	u := fetchFromDb(id)

	// Check if name was provided in the input
	if v, isSet := changes["name"]; isSet { // v is the value with type `interface{}`
		value, valid := v.(*string)  // *string, could be nil
		if !valid {
			// map values are automatically coerced to the types defined in the schema, 
			// so if this error is thrown it's most likely a type mismatch between here and your GraphQL input definition
			return nil, errors.New("field 'name' on UserChanges does not have type String")
		}

		if value == nil {
			u.Name = "" // value to use when null
		} else {
			u.Name = *value // set to the provided value
		}
	}
	// If !isSet, the field was omitted entirely - no change

	// Alternative: use reflection (see below)
	if err := ApplyChanges(changes, &u); err != nil {
		return nil, err
	}
	
	saveToDb(u)
	return u, nil
}
```

Please note that map values are automatically coerced to the types defined in the schema.
This means that optional, nested inputs or scalars will conform to their expected types.

We often use the mapstructure library to directly apply these changesets directly to the object using reflection:
```go

func ApplyChanges(changes map[string]interface{}, to interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		TagName:     "json",
		Result:      to,
		ZeroFields:  true,
		// This is needed to get mapstructure to call the gqlgen unmarshaler func for custom scalars (eg Date)
		DecodeHook: func(a reflect.Type, b reflect.Type, v interface{}) (interface{}, error) {
			if reflect.PtrTo(b).Implements(reflect.TypeOf((*graphql.Unmarshaler)(nil)).Elem()) {
				resultType := reflect.New(b)
				result := resultType.MethodByName("UnmarshalGQL").Call([]reflect.Value{reflect.ValueOf(v)})
				err, _ := result[0].Interface().(error)
				return resultType.Elem().Interface(), err
			}

			return v, nil
		},
	})

	if err != nil {
		return err
	}

	return dec.Decode(changes)
}
```

## Omittable

The `Omittable[T]` type provides a more type-safe alternative to maps for distinguishing between unset, null, and actual values. It's a generic wrapper that tracks both the value and whether it was explicitly provided.

You can enable omittable fields in two ways:

**Option 1: Per-field with directive**
```graphql
input UserChanges {
	name: String @goField(omittable: true)
	email: String @goField(omittable: true)
}
```

**Option 2: Globally in config**
```yaml
# gqlgen.yml
nullable_input_omittable: true
```

This generates a Go struct using `graphql.Omittable`:

```go
type UserChanges struct {
	Name  graphql.Omittable[*string] `json:"name,omitempty"`
	Email graphql.Omittable[*string] `json:"email,omitempty"`
}
```

Your resolver can then distinguish between three states:

```go
func (r *mutationResolver) UpdateUser(ctx context.Context, id int, changes UserChanges) (*User, error) {
	u := fetchFromDb(id)
	
	// Check if name was provided in the input
	if changes.Name.IsSet() {
		value := changes.Name.Value() // *string, could be nil
		if value == nil {
			u.Name = "" // value to use when null
		} else {
			u.Name = *value // set to the provided value
		}
	}
	// If !changes.Name.IsSet(), the field was omitted entirely - no change
	
	// Alternative: use ValueOK for cleaner code
	if value, isSet := changes.Email.ValueOK(); isSet {
		u.Email = value // *string, nil if null was provided, actual value otherwise
	}
	
	saveToDb(u)
	return u, nil
}
```

### Key Methods

- `IsSet()` - Returns true if the field was explicitly provided (even if null)
- `Value()` - Returns the value, or zero value if not set
- `ValueOK()` - Returns (value, wasSet) similar to map access
- `OmittableOf(value)` - Helper to create an Omittable with a value

