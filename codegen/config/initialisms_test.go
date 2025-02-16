package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/templates"
)

func TestGoInitialismsConfig(t *testing.T) {
	t.Run("load go initialisms config", func(t *testing.T) {
		config, err := LoadConfig("testdata/cfg/goInitialisms.yml")
		require.NoError(t, err)
		require.True(t, config.GoInitialisms.ReplaceDefaults)
		require.Len(t, config.GoInitialisms.Initialisms, 2)
	})
	t.Run("empty initialism config doesn't change anything", func(t *testing.T) {
		tt := GoInitialismsConfig{}
		result := tt.determineGoInitialisms()
		assert.Equal(t, len(templates.CommonInitialisms), len(result))
	})
	t.Run("initialism config appends if desired", func(t *testing.T) {
		tt := GoInitialismsConfig{ReplaceDefaults: false, Initialisms: []string{"ASDF"}}
		result := tt.determineGoInitialisms()
		assert.Len(t, result, len(templates.CommonInitialisms)+1)
		assert.True(t, result["ASDF"])
	})
	t.Run("initialism config replaces if desired", func(t *testing.T) {
		tt := GoInitialismsConfig{ReplaceDefaults: true, Initialisms: []string{"ASDF"}}
		result := tt.determineGoInitialisms()
		assert.Len(t, result, 1)
		assert.True(t, result["ASDF"])
	})
	t.Run("initialism config uppercases the initialisms", func(t *testing.T) {
		tt := GoInitialismsConfig{Initialisms: []string{"asdf"}}
		result := tt.determineGoInitialisms()
		assert.True(t, result["ASDF"])
	})
}
