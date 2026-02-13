package config

import (
	"fmt"
	"sync"
	"sync/atomic"
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
		assert.Len(t, result, len(templates.CommonInitialisms))
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

func TestGoInitialismsConcurrentSetAndRead(t *testing.T) {
	t.Cleanup(func() {
		templates.SetInitialisms(templates.CommonInitialisms)
	})

	const workers = 8
	const iterations = 128

	start := make(chan struct{})
	var wg sync.WaitGroup
	var emptyReads atomic.Int32

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				GoInitialismsConfig{
					ReplaceDefaults: true,
					Initialisms:     []string{fmt.Sprintf("worker_%d_%d", i, j)},
				}.setInitialisms()
			}
		}(i)
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				if len(templates.GetInitialisms()) == 0 {
					emptyReads.Add(1)
				}
			}
		}()
	}

	close(start)
	wg.Wait()

	assert.Zero(t, emptyReads.Load())
}
