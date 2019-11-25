---
title: "Allowing mutation of generated models before rendering"
description: How to use a model mutation function to insert a ORM-specific tags onto struct fields.
linkTitle: "Modelgen hook"
menu: { main: { parent: 'recipes' } }
---

The following recipe shows how to use a `modelgen` plugin hook to mutate generated
models before they are rendered into a resulting file. This feature has many uses but
the example focuses only on inserting ORM-specific tags into generated struct fields. This
is a common use case since it allows for better field matching of DB queries and
the generated data structure.

First of all, we need to create a function that will mutate the generated model.
Then we can attach the function to the plugin and use it like any other plugin.

``` go
import (
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/modelgen"
)

// Defining mutation function
func mutateHook(b *modelgen.ModelBuild) *modelgen.ModelBuild {
	for _, model := range b.Models {
		for _, field := range model.Fields {
			field.Tag += ` orm_binding:"` + model.Name + `.` +  field.Name + `"`
		}
	}

	return b
}

func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}

	// Attaching the mutation function onto modelgen plugin
	p := modelgen.Plugin{
		MutateHook: mutateHook,
	}

	err = api.Generate(cfg,
		api.NoPlugins(),
		api.AddPlugin(&p),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
```

Now fields from generated models will contain a additional tag `orm_binding`.

This schema:

```graphql
type Object {
    field1: String
    field2: Int
}
```

Will gen generated into:

```go
type Object struct {
	field1 *string  `json:"field1" orm_binding:"Object.field1"`
	field2 *int     `json:"field2" orm_binding:"Object.field2"`
}
```
