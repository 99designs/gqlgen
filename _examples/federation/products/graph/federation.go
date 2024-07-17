// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graph

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/99designs/gqlgen/plugin/federation/fedruntime"
)

var (
	ErrUnknownType  = errors.New("unknown type")
	ErrTypeNotFound = errors.New("type not found")
)

func (ec *executionContext) __resolve__service(ctx context.Context) (fedruntime.Service, error) {
	if ec.DisableIntrospection {
		return fedruntime.Service{}, errors.New("federated introspection disabled")
	}

	var sdl []string

	for _, src := range sources {
		if src.BuiltIn {
			continue
		}
		sdl = append(sdl, src.Input)
	}

	return fedruntime.Service{
		SDL: strings.Join(sdl, "\n"),
	}, nil
}

func (ec *executionContext) __resolve_entities(ctx context.Context, representations []map[string]interface{}) []fedruntime.Entity {
	list := make([]fedruntime.Entity, len(representations))

	repsMap := ec.buildRepresentationGroups(ctx, representations)

	switch len(repsMap) {
	case 0:
		return list
	case 1:
		for typeName, reps := range repsMap {
			ec.resolveEntityGroup(ctx, typeName, reps.entityRepresentations, reps.indexes, list)
		}
		return list
	default:
		var g sync.WaitGroup
		g.Add(len(repsMap))
		for typeName, reps := range repsMap {
			go func(typeName string, reps []EntityRepresentation, idx []int) {
				ec.resolveEntityGroup(ctx, typeName, reps, idx, list)
				g.Done()
			}(typeName, reps.entityRepresentations, reps.indexes)
		}
		g.Wait()
		return list
	}
}

type GroupedRepresentations struct {
	indexes               []int
	entityRepresentations []EntityRepresentation
}

// EntityRepresentation is the JSON representation of an entity sent by the Router
// used as the inputs for us to resolve.
//
// We make it a map because we know the top level JSON is always an object.
type EntityRepresentation map[string]any

// We group entities by typename so that we can parallelize their resolution.
// This is particularly helpful when there are entity groups in multi mode.
func (ec *executionContext) buildRepresentationGroups(
	ctx context.Context,
	representations []map[string]any,
) map[string]GroupedRepresentations {
	repsMap := make(map[string]GroupedRepresentations)
	for i, rep := range representations {
		typeName, ok := rep["__typename"].(string)
		if !ok {
			// If there is no __typename, we just skip the representation;
			// we just won't be resolving these unknown types.
			ec.Error(ctx, errors.New("__typename must be an existing string"))
			continue
		}

		groupedRepresentations := repsMap[typeName]
		groupedRepresentations.indexes = append(groupedRepresentations.indexes, i)
		groupedRepresentations.entityRepresentations = append(groupedRepresentations.entityRepresentations, rep)
		repsMap[typeName] = groupedRepresentations
	}

	return repsMap
}

func (ec *executionContext) resolveEntityGroup(
	ctx context.Context,
	typeName string,
	reps []EntityRepresentation,
	idx []int,
	list []fedruntime.Entity,
) {
	if isMulti(typeName) {
		err := ec.resolveManyEntities(ctx, typeName, reps, idx, list)
		if err != nil {
			ec.Error(ctx, err)
		}
	} else {
		// if there are multiple entities to resolve, parallelize (similar to
		// graphql.FieldSet.Dispatch)
		var e sync.WaitGroup
		e.Add(len(reps))
		for i, rep := range reps {
			i, rep := i, rep
			go func(i int, rep EntityRepresentation) {
				err := ec.resolveEntity(ctx, typeName, rep, idx, i, list)
				if err != nil {
					ec.Error(ctx, err)
				}
				e.Done()
			}(i, rep)
		}
		e.Wait()
	}
}

func isMulti(typeName string) bool {
	switch typeName {
	default:
		return false
	}
}

func (ec *executionContext) resolveEntity(
	ctx context.Context,
	typeName string,
	rep EntityRepresentation,
	idx []int, i int,
	list []fedruntime.Entity,
) (err error) {
	// we need to do our own panic handling, because we may be called in a
	// goroutine, where the usual panic handling can't catch us
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
		}
	}()

	switch typeName {
	case "Manufacturer":
		resolverName, err := entityResolverNameForManufacturer(ctx, rep)
		if err != nil {
			return fmt.Errorf(`finding resolver for Entity "Manufacturer": %w`, err)
		}
		switch resolverName {

		case "findManufacturerByID":
			id0, err := ec.unmarshalNString2string(ctx, rep["id"])
			if err != nil {
				return fmt.Errorf(`unmarshalling param 0 for findManufacturerByID(): %w`, err)
			}
			entity, err := ec.resolvers.Entity().FindManufacturerByID(ctx, id0)
			if err != nil {
				return fmt.Errorf(`resolving Entity "Manufacturer": %w`, err)
			}

			list[idx[i]] = entity
			return nil
		}
	case "Product":
		resolverName, err := entityResolverNameForProduct(ctx, rep)
		if err != nil {
			return fmt.Errorf(`finding resolver for Entity "Product": %w`, err)
		}
		switch resolverName {

		case "findProductByManufacturerIDAndID":
			id0, err := ec.unmarshalNString2string(ctx, rep["manufacturer"].(EntityRepresentation)["id"])
			if err != nil {
				return fmt.Errorf(`unmarshalling param 0 for findProductByManufacturerIDAndID(): %w`, err)
			}
			id1, err := ec.unmarshalNString2string(ctx, rep["id"])
			if err != nil {
				return fmt.Errorf(`unmarshalling param 1 for findProductByManufacturerIDAndID(): %w`, err)
			}
			entity, err := ec.resolvers.Entity().FindProductByManufacturerIDAndID(ctx, id0, id1)
			if err != nil {
				return fmt.Errorf(`resolving Entity "Product": %w`, err)
			}

			list[idx[i]] = entity
			return nil
		case "findProductByUpc":
			id0, err := ec.unmarshalNString2string(ctx, rep["upc"])
			if err != nil {
				return fmt.Errorf(`unmarshalling param 0 for findProductByUpc(): %w`, err)
			}
			entity, err := ec.resolvers.Entity().FindProductByUpc(ctx, id0)
			if err != nil {
				return fmt.Errorf(`resolving Entity "Product": %w`, err)
			}

			list[idx[i]] = entity
			return nil
		}

	}
	return fmt.Errorf("%w: %s", ErrUnknownType, typeName)
}

func (ec *executionContext) resolveManyEntities(
	ctx context.Context,
	typeName string,
	reps []EntityRepresentation,
	idx []int,
	list []fedruntime.Entity,
) (err error) {
	// we need to do our own panic handling, because we may be called in a
	// goroutine, where the usual panic handling can't catch us
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
		}
	}()

	switch typeName {

	default:
		return errors.New("unknown type: " + typeName)
	}
}

func entityResolverNameForManufacturer(ctx context.Context, rep EntityRepresentation) (string, error) {
	for {
		var (
			m   EntityRepresentation
			val interface{}
			ok  bool
		)
		_ = val
		// if all of the KeyFields values for this resolver are null,
		// we shouldn't use use it
		allNull := true
		m = rep
		val, ok = m["id"]
		if !ok {
			break
		}
		if allNull {
			allNull = val == nil
		}
		if allNull {
			break
		}
		return "findManufacturerByID", nil
	}
	return "", fmt.Errorf("%w for Manufacturer", ErrTypeNotFound)
}

func entityResolverNameForProduct(ctx context.Context, rep EntityRepresentation) (string, error) {
	for {
		var (
			m   EntityRepresentation
			val interface{}
			ok  bool
		)
		_ = val
		// if all of the KeyFields values for this resolver are null,
		// we shouldn't use use it
		allNull := true
		m = rep
		val, ok = m["manufacturer"]
		if !ok {
			break
		}
		if m, ok = val.(map[string]interface{}); !ok {
			break
		}
		val, ok = m["id"]
		if !ok {
			break
		}
		if allNull {
			allNull = val == nil
		}
		m = rep
		val, ok = m["id"]
		if !ok {
			break
		}
		if allNull {
			allNull = val == nil
		}
		if allNull {
			break
		}
		return "findProductByManufacturerIDAndID", nil
	}
	for {
		var (
			m   EntityRepresentation
			val interface{}
			ok  bool
		)
		_ = val
		// if all of the KeyFields values for this resolver are null,
		// we shouldn't use use it
		allNull := true
		m = rep
		val, ok = m["upc"]
		if !ok {
			break
		}
		if allNull {
			allNull = val == nil
		}
		if allNull {
			break
		}
		return "findProductByUpc", nil
	}
	return "", fmt.Errorf("%w for Product", ErrTypeNotFound)
}
