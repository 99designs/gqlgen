package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntityObjectDirectives validates that OBJECT-level directives
// are correctly identified and filtered on federated entities.
func TestEntityObjectDirectives(t *testing.T) {
	t.Run("entities_identified_with_directives", func(t *testing.T) {
		f, _ := load(t, "testdata/entitydirectives/gqlgen.yml")

		// Verify entities are created with @key directives
		require.Len(t, f.Entities, 3, "should have 3 entities with @key directives")

		// Find entities
		var personEntity *Entity
		var productEntity *Entity
		var basicEntity *Entity
		for _, e := range f.Entities {
			switch e.Name {
			case "Person":
				personEntity = e
			case "Product":
				productEntity = e
			case "Basic":
				basicEntity = e
			}
		}

		require.NotNil(t, personEntity, "Person entity should exist")
		require.NotNil(t, productEntity, "Product entity should exist")
		require.NotNil(t, basicEntity, "Basic entity should exist")

		// Verify NonFederationDirectives filters correctly at AST level
		// Person has @key and @guard, should only return @guard
		personNonFedDirectives := personEntity.NonFederationDirectives()
		require.Len(t, personNonFedDirectives, 1, "Person should have 1 non-federation directive")
		assert.Equal(t, "guard", personNonFedDirectives[0].Name)

		// Product has @key, @auth, and @guard, should return @auth and @guard
		productNonFedDirectives := productEntity.NonFederationDirectives()
		require.Len(
			t,
			productNonFedDirectives,
			2,
			"Product should have 2 non-federation directives",
		)
		directiveNames := []string{productNonFedDirectives[0].Name, productNonFedDirectives[1].Name}
		assert.Contains(t, directiveNames, "guard")
		assert.Contains(t, directiveNames, "auth")

		// Basic has only @key, should have no non-federation directives
		basicNonFedDirectives := basicEntity.NonFederationDirectives()
		assert.Nil(t, basicNonFedDirectives, "Basic should have no non-federation directives")
	})

	t.Run("federation_directives_excluded", func(t *testing.T) {
		// Verify that federation directives are correctly filtered
		testCases := []string{
			"key", "requires", "provides", "external", "extends",
			"tag", "inaccessible", "shareable", "interfaceObject",
			"override", "composeDirective", "link",
			"authenticated", "requiresScopes", "policy",
			"entityResolver", "goModel", "goField", "goTag",
		}

		for _, name := range testCases {
			assert.True(t, federationDirectiveNames[name],
				"federation directive %q should be in exclusion list", name)
		}
	})

	t.Run("non_federation_directives_preserved", func(t *testing.T) {
		f, cfg := load(t, "testdata/entitydirectives/gqlgen.yml")
		require.NoError(t, f.MutateConfig(cfg))

		// Find Person entity and check AST-level directives
		var personEntity *Entity
		for _, e := range f.Entities {
			if e.Name == "Person" {
				personEntity = e
				break
			}
		}

		require.NotNil(t, personEntity)

		// NonFederationDirectives should return only @guard (not @key)
		nonFedDirectives := personEntity.NonFederationDirectives()
		require.Len(t, nonFedDirectives, 1)
		assert.Equal(t, "guard", nonFedDirectives[0].Name)

		// Verify @key is filtered out
		allDirectives := personEntity.Def.Directives
		hasKey := false
		for _, d := range allDirectives {
			if d.Name == "key" {
				hasKey = true
				break
			}
		}
		assert.True(t, hasKey, "Person should have @key in full directive list")
	})
}
