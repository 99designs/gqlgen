package codegen

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/neelance/schema"
	"gopkg.in/yaml.v2"
)

var defaults = Config{
	SchemaFilename: "schema.graphql",
	Model:          PackageConfig{Filename: "models_gen.go"},
	Exec:           PackageConfig{Filename: "generated.go"},
}

var cfgFilenames = []string{".gqlgen.yml", "gqlgen.yml", "gqlgen.yaml"}

// LoadDefaultConfig looks for a config file in the current directory, and all parent directories
// walking up the tree. The closest config file will be returned.
func LoadDefaultConfig() (*Config, error) {
	cfgFile, err := findCfg()
	if err != nil || cfgFile == "" {
		cpy := defaults
		return &cpy, err
	}

	err = os.Chdir(filepath.Dir(cfgFile))
	if err != nil {
		return nil, errors.Wrap(err, "unable to enter config dir")
	}
	return LoadConfig(cfgFile)
}

// LoadConfig reads the gqlgen.yml config file
func LoadConfig(filename string) (*Config, error) {
	config := defaults

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read config")
	}

	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}

	return &config, nil
}

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
	// If Package is not set, first attempt to load the package at the output dir. If that fails
	// fallback to just the base dir name of the output filename.
	if c.Package == "" {
		cwd, _ := os.Getwd()
		pkg, err := build.Default.Import(c.Dir(), cwd, 0)
		if err != nil {
			c.Package = filepath.Base(c.Dir())
		} else {
			c.Package = pkg.Name
		}
	}
	c.Package = sanitizePackageName(c.Package)
	c.Filename = abs(c.Filename)
	return nil
}

func (c *PackageConfig) ImportPath() string {
	return importPath(c.Dir())
}

func (c *PackageConfig) Dir() string {
	return filepath.ToSlash(filepath.Dir(c.Filename))
}

func (c *PackageConfig) Check() error {
	if strings.ContainsAny(c.Package, "./\\") {
		return fmt.Errorf("package should be the output package name only, do not include the output filename")
	}
	if c.Filename != "" && !strings.HasSuffix(c.Filename, ".go") {
		return fmt.Errorf("filename should be path to a go source file")
	}
	return nil
}

func (cfg *Config) Check() error {
	if err := cfg.Models.Check(); err != nil {
		return errors.Wrap(err, "config.models")
	}
	if err := cfg.Exec.Check(); err != nil {
		return errors.Wrap(err, "config.exec")
	}
	if err := cfg.Model.Check(); err != nil {
		return errors.Wrap(err, "config.model")
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
		if strings.LastIndex(entry.Model, ".") < strings.LastIndex(entry.Model, "/") {
			return fmt.Errorf("model %s: invalid type specifier \"%s\" - you need to specify a struct to map to", typeName, entry.Model)
		}
	}
	return nil
}

// findCfg searches for the config file in this directory and all parents up the tree
// looking for the closest match
func findCfg() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get working dir to findCfg")
	}

	cfg := findCfgInDir(dir)
	for cfg == "" && dir != "/" {
		dir = filepath.Dir(dir)
		cfg = findCfgInDir(dir)
	}

	return cfg, nil
}

func findCfgInDir(dir string) string {
	for _, cfgName := range cfgFilenames {
		path := filepath.Join(dir, cfgName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
