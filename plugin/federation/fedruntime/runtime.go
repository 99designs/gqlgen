package fedruntime

import (
	"errors"
	"fmt"
)

// Service is the service object that the
// generated.go file will return for the _service
// query
type Service struct {
	SDL string `json:"sdl"`
}

// Everything with a @key implements this
type Entity interface {
	IsEntity()
}

// Used for the Link directive
type Link any

var (
	// ErrUnknownType is returned when an unknown entity type is encountered
	ErrUnknownType = errors.New("unknown type")
	// ErrTypeNotFound is returned when an entity type cannot be resolved
	ErrTypeNotFound = errors.New("type not found")
)

// KeyFieldCheck represents a key field validation check.
type KeyFieldCheck struct {
	// FieldPath is the path to the field (e.g., ["id"] or ["user", "id"] for nested fields)
	FieldPath []string
}

// ResolverKeyCheck represents the key requirements for a resolver.
type ResolverKeyCheck struct {
	// ResolverName is the name of the resolver function
	ResolverName string
	// KeyFields are the required key fields for this resolver
	KeyFields []KeyFieldCheck
}

// ValidateEntityKeys determines which resolver to use for an entity representation.
// It checks that all required key fields exist and are not all null.
// Returns the resolver name if valid, or an error if no resolver matches.
func ValidateEntityKeys(
	entityName string,
	rep map[string]any,
	resolverChecks []ResolverKeyCheck,
) (string, error) {
	var allErrors []error

	for _, resolverCheck := range resolverChecks {
		if err := validateResolverKeys(entityName, rep, resolverCheck); err != nil {
			allErrors = append(allErrors, err)
			continue
		}
		// Found a valid resolver
		return resolverCheck.ResolverName, nil
	}

	// No valid resolver found
	if len(allErrors) > 0 {
		return "", fmt.Errorf("%w for %s due to %v",
			ErrTypeNotFound, entityName, errors.Join(allErrors...))
	}
	return "", fmt.Errorf("%w for %s: no resolvers defined", ErrTypeNotFound, entityName)
}

// validateResolverKeys checks if a resolver's key fields are valid.
func validateResolverKeys(entityName string, rep map[string]any, check ResolverKeyCheck) error {
	allNull := true

	for _, keyField := range check.KeyFields {
		val, err := getNestedField(rep, keyField.FieldPath)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTypeNotFound, err)
		}

		if val != nil {
			allNull = false
		}
	}

	if allNull {
		return fmt.Errorf("%w due to all null value KeyFields for %s",
			ErrTypeNotFound, entityName)
	}

	return nil
}

// getNestedField retrieves a value from a nested map by following a field path.
func getNestedField(rep map[string]any, path []string) (any, error) {
	if len(path) == 0 {
		return nil, errors.New("empty field path")
	}

	current := rep
	for i, fieldName := range path {
		val, ok := current[fieldName]
		if !ok {
			return nil, fmt.Errorf("missing Key Field %q", fieldName)
		}

		// If this is not the last field in the path, it should be a map
		if i < len(path)-1 {
			nextMap, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf(
					"nested Key Field %q value not matching map[string]any",
					fieldName,
				)
			}
			current = nextMap
		} else {
			// Last field - return its value
			return val, nil
		}
	}

	return nil, errors.New("unexpected: empty path processed")
}
