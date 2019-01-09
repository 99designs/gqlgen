package config

import (
	"go/types"

	"golang.org/x/tools/go/loader"
)

func (c *Config) NewLoaderWithErrors() loader.Config {
	conf := loader.Config{}

	for _, pkg := range c.Models.ReferencedPackages() {
		conf.Import(pkg)
	}
	return conf
}

func (c *Config) NewLoaderWithoutErrors() loader.Config {
	conf := c.NewLoaderWithErrors()
	conf.AllowErrors = true
	conf.TypeChecker = types.Config{
		Error: func(e error) {},
	}
	return conf
}
