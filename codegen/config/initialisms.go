package config

import (
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
)

// GoInitialismsConfig allows to modify the default behavior of naming Go methods, types and properties
type GoInitialismsConfig struct {
	// If true, the Initialisms won't get appended to the default ones but replace them
	ReplaceDefaults bool `yaml:"replace_defaults"`
	// Custom initialisms to be added or to replace the default ones
	Initialisms []string `yaml:"initialisms"`
}

// setInitialisms adjustes GetInitialisms based on its settings.
func (i GoInitialismsConfig) setInitialisms() {
	toUse := i.determineGoInitialisms()
	templates.GetInitialisms = func() map[string]bool {
		return toUse
	}
}

// determineGoInitialisms returns the Go initialims to be used, based on its settings.
func (i GoInitialismsConfig) determineGoInitialisms() (initialismsToUse map[string]bool) {
	if i.ReplaceDefaults {
		initialismsToUse = make(map[string]bool, len(i.Initialisms))
		for _, initialism := range i.Initialisms {
			initialismsToUse[strings.ToUpper(initialism)] = true
		}
	} else {
		initialismsToUse = make(map[string]bool, len(templates.CommonInitialisms)+len(i.Initialisms))
		for initialism, value := range templates.CommonInitialisms {
			initialismsToUse[strings.ToUpper(initialism)] = value
		}
		for _, initialism := range i.Initialisms {
			initialismsToUse[strings.ToUpper(initialism)] = true
		}
	}
	return initialismsToUse
}
