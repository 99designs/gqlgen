package generated

import (
	"context"
	"encoding/json"
)

// PopulateMultiPlanetRequires is the requires populator for the MultiPlanet entity.
func (ec *executionContext) PopulateMultiPlanetRequires(ctx context.Context, entity *MultiPlanet, reps map[string]any) error {
	diameter, err := reps["diameter"].(json.Number).Int64()
	if err != nil {
		return err
	}
	entity.Diameter = int(diameter)
	return nil
}

// PopulatePlanetRequires is the requires populator for the Planet entity.
func (ec *executionContext) PopulatePlanetRequires(ctx context.Context, entity *Planet, reps map[string]any) error {
	diameter, err := reps["diameter"].(json.Number).Int64()
	if err != nil {
		return err
	}
	entity.Diameter = int(diameter)
	return nil
}
