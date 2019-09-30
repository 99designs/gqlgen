package plugin

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
)

func ConfigurePlugins(cfg *config.Config, plugins []Plugin) error {
	var seen = map[string]bool{}
	for _, p := range plugins {
		if seen[p.Name()] {
			return errors.Errorf("plugin %q already registered", p.Name())
		}
		if err := cfg.ConfigurePlugin(p.Name(), p); err != nil {
			return errors.Wrapf(err, "unable to configure plugin %q", p.Name())
		}
		seen[p.Name()] = true
	}
	return nil
}
