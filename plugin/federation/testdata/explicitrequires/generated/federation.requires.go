package generated

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/99designs/gqlgen/plugin/federation/testdata/explicitrequires/generated/model"
)

// PopulateMultiHelloMultipleRequiresRequires is the requires populator for the MultiHelloMultipleRequires entity.
func (ec *executionContext) PopulateMultiHelloMultipleRequiresRequires(ctx context.Context, entity *model.MultiHelloMultipleRequires, reps map[string]interface{}) error {
	panic(fmt.Errorf("not implemented: PopulateMultiHelloMultipleRequiresRequires"))
}

// PopulateMultiHelloRequiresRequires is the requires populator for the MultiHelloRequires entity.
func (ec *executionContext) PopulateMultiHelloRequiresRequires(ctx context.Context, entity *model.MultiHelloRequires, reps map[string]interface{}) error {
	panic(fmt.Errorf("not implemented: PopulateMultiHelloRequiresRequires"))
}

// PopulateMultiPlanetRequiresNestedRequires is the requires populator for the MultiPlanetRequiresNested entity.
func (ec *executionContext) PopulateMultiPlanetRequiresNestedRequires(ctx context.Context, entity *model.MultiPlanetRequiresNested, reps map[string]interface{}) error {
	panic(fmt.Errorf("not implemented: PopulateMultiPlanetRequiresNestedRequires"))
}

// PopulatePlanetMultipleRequiresRequires is the requires populator for the PlanetMultipleRequires entity.
func (ec *executionContext) PopulatePlanetMultipleRequiresRequires(ctx context.Context, entity *model.PlanetMultipleRequires, reps map[string]interface{}) error {
	diameter, _ := reps["diameter"].(json.Number).Int64()
	density, _ := reps["density"].(json.Number).Int64()
	entity.Name = reps["name"].(string)
	entity.Diameter = int(diameter)
	entity.Density = int(density)
	return nil
}

// PopulatePlanetRequiresNestedRequires is the requires populator for the PlanetRequiresNested entity.
func (ec *executionContext) PopulatePlanetRequiresNestedRequires(ctx context.Context, entity *model.PlanetRequiresNested, reps map[string]interface{}) error {
	entity.Name = reps["name"].(string)
	entity.World = &model.World{
		Foo: reps["world"].(map[string]interface{})["foo"].(string),
	}
	return nil
}

// PopulatePlanetRequiresRequires is the requires populator for the PlanetRequires entity.
func (ec *executionContext) PopulatePlanetRequiresRequires(ctx context.Context, entity *model.PlanetRequires, reps map[string]interface{}) error {
	diameter, _ := reps["diameter"].(json.Number).Int64()
	entity.Name = reps["name"].(string)
	entity.Diameter = int(diameter)
	return nil
}
