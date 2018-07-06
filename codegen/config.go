package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/neelance/schema"
)

type Config struct {
	SchemaFilename string        `yaml:"schema,omitempty"`
	SchemaStr      string        `yaml:"-"`
	Exec           PackageConfig `yaml:"exec"`
	Model          PackageConfig `yaml:"model"`
	Models         TypeMap       `yaml:"models,omitempty"`

	schema *schema.Schema `yaml:"-"`
}

type PackageConfig struct {
	Filename string `yaml:"filename,omitempty"`
	Package  string `yaml:"package,omitempty"`
}

type TypeMapEntry struct {
	Model  string                  `yaml:"model"`
	Fields map[string]TypeMapField `yaml:"fields,omitempty"`
}

type TypeMapField struct {
	Resolver bool `yaml:"resolver"`
}

func (c *PackageConfig) normalize() error {
	if c.Filename == "" {
		return errors.New("Filename is required")
	}
	c.Filename = abs(c.Filename)
	if c.Package == "" {
		c.Package = filepath.Base(c.Dir())
	}
	c.Package = sanitizePackageName(c.Package)
	return nil
}

func (c *PackageConfig) ImportPath() string {
	return importPath(c.Dir(), c.Package)
}

func (c *PackageConfig) Dir() string {
	return filepath.ToSlash(filepath.Dir(c.Filename))
}

func (cfg *Config) Check() error {
	err := cfg.Models.Check()
	if err != nil {
		return fmt.Errorf("config: %s", err.Error())
	}
	return nil
}

type TypeMap map[string]TypeMapEntry

func (tm TypeMap) Exists(typeName string) bool {
	_, ok := tm[typeName]
	return ok
}

func (tm TypeMap) Check() error {
	for typeName, entry := range tm {
		if entry.Model == "" {
			return fmt.Errorf("model %s: entityPath is not defined", typeName)
		}
	}
	return nil
}
