package config

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/internal/gopath"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"gopkg.in/yaml.v2"
)

type Config struct {
	SchemaFilename SchemaFilenames `yaml:"schema,omitempty"`
	Exec           PackageConfig   `yaml:"exec"`
	Model          PackageConfig   `yaml:"model"`
	Resolver       PackageConfig   `yaml:"resolver,omitempty"`
	Models         TypeMap         `yaml:"models,omitempty"`
	StructTag      string          `yaml:"struct_tag,omitempty"`
}

var cfgFilenames = []string{".gqlgen.yml", "gqlgen.yml", "gqlgen.yaml"}

// DefaultConfig creates a copy of the default config
func DefaultConfig() *Config {
	return &Config{
		SchemaFilename: SchemaFilenames{"schema.graphql"},
		Model:          PackageConfig{Filename: "models_gen.go"},
		Exec:           PackageConfig{Filename: "generated.go"},
	}
}

// LoadConfigFromDefaultLocations looks for a config file in the current directory, and all parent directories
// walking up the tree. The closest config file will be returned.
func LoadConfigFromDefaultLocations() (*Config, error) {
	cfgFile, err := findCfg()
	if err != nil {
		return nil, err
	}

	err = os.Chdir(filepath.Dir(cfgFile))
	if err != nil {
		return nil, errors.Wrap(err, "unable to enter config dir")
	}
	return LoadConfig(cfgFile)
}

// LoadConfig reads the gqlgen.yml config file
func LoadConfig(filename string) (*Config, error) {
	config := DefaultConfig()

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read config")
	}

	if err := yaml.UnmarshalStrict(b, config); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}

	preGlobbing := config.SchemaFilename
	config.SchemaFilename = SchemaFilenames{}
	for _, f := range preGlobbing {
		matches, err := filepath.Glob(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to glob schema filename %s", f)
		}

		for _, m := range matches {
			if config.SchemaFilename.Has(m) {
				continue
			}
			config.SchemaFilename = append(config.SchemaFilename, m)
		}
	}

	return config, nil
}

type PackageConfig struct {
	Filename string `yaml:"filename,omitempty"`
	Package  string `yaml:"package,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

type TypeMapEntry struct {
	Model  string                  `yaml:"model"`
	Fields map[string]TypeMapField `yaml:"fields,omitempty"`
}

type TypeMapField struct {
	Resolver  bool   `yaml:"resolver"`
	FieldName string `yaml:"fieldName"`
}

type SchemaFilenames []string

func (a *SchemaFilenames) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	err := unmarshal(&single)
	if err == nil {
		*a = []string{single}
		return nil
	}

	var multi []string
	err = unmarshal(&multi)
	if err != nil {
		return err
	}

	*a = multi
	return nil
}

func (a SchemaFilenames) Has(file string) bool {
	for _, existing := range a {
		if existing == file {
			return true
		}
	}
	return false
}

func (c *PackageConfig) normalize() error {
	if c.Filename == "" {
		return errors.New("Filename is required")
	}
	c.Filename = abs(c.Filename)
	// If Package is not set, first attempt to load the package at the output dir. If that fails
	// fallback to just the base dir name of the output filename.
	if c.Package == "" {
		cwd, _ := os.Getwd()
		pkg, _ := build.Default.Import(c.ImportPath(), cwd, 0)
		if pkg.Name != "" {
			c.Package = pkg.Name
		} else {
			c.Package = filepath.Base(c.Dir())
		}
	}
	c.Package = sanitizePackageName(c.Package)
	return nil
}

func (c *PackageConfig) ImportPath() string {
	return gopath.MustDir2Import(c.Dir())
}

func (c *PackageConfig) Dir() string {
	return filepath.Dir(c.Filename)
}

func (c *PackageConfig) Check() error {
	if strings.ContainsAny(c.Package, "./\\") {
		return fmt.Errorf("package should be the output package name only, do not include the output filename")
	}
	if c.Filename != "" && !strings.HasSuffix(c.Filename, ".go") {
		return fmt.Errorf("filename should be path to a go source file")
	}

	return c.normalize()
}

func (c *PackageConfig) IsDefined() bool {
	return c.Filename != ""
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
	if cfg.Resolver.IsDefined() {
		if err := cfg.Resolver.Check(); err != nil {
			return errors.Wrap(err, "config.resolver")
		}
	}

	return cfg.normalize()
}

type TypeMap map[string]TypeMapEntry

func (tm TypeMap) Exists(typeName string) bool {
	_, ok := tm[typeName]
	return ok
}

func (tm TypeMap) Check() error {
	for typeName, entry := range tm {
		if strings.LastIndex(entry.Model, ".") < strings.LastIndex(entry.Model, "/") {
			return fmt.Errorf("model %s: invalid type specifier \"%s\" - you need to specify a struct to map to", typeName, entry.Model)
		}
	}
	return nil
}

func (tm TypeMap) ReferencedPackages() []string {
	var pkgs []string

	for _, typ := range tm {
		if typ.Model == "map[string]interface{}" {
			continue
		}
		pkg, _ := gopath.PkgAndType(typ.Model)
		if pkg == "" || inStrSlice(pkgs, pkg) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i] > pkgs[j]
	})
	return pkgs
}

func inStrSlice(haystack []string, needle string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}

	return false
}

// findCfg searches for the config file in this directory and all parents up the tree
// looking for the closest match
func findCfg() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get working dir to findCfg")
	}

	cfg := findCfgInDir(dir)

	for cfg == "" && dir != filepath.Dir(dir) {
		dir = filepath.Dir(dir)
		cfg = findCfgInDir(dir)
	}

	if cfg == "" {
		return "", os.ErrNotExist
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

func (cfg *Config) normalize() error {
	if err := cfg.Model.normalize(); err != nil {
		return errors.Wrap(err, "model")
	}

	if err := cfg.Exec.normalize(); err != nil {
		return errors.Wrap(err, "exec")
	}

	if cfg.Resolver.IsDefined() {
		if err := cfg.Resolver.normalize(); err != nil {
			return errors.Wrap(err, "resolver")
		}
	}

	builtins := TypeMap{
		"__Directive":  {Model: "github.com/99designs/gqlgen/graphql/introspection.Directive"},
		"__Type":       {Model: "github.com/99designs/gqlgen/graphql/introspection.Type"},
		"__Field":      {Model: "github.com/99designs/gqlgen/graphql/introspection.Field"},
		"__EnumValue":  {Model: "github.com/99designs/gqlgen/graphql/introspection.EnumValue"},
		"__InputValue": {Model: "github.com/99designs/gqlgen/graphql/introspection.InputValue"},
		"__Schema":     {Model: "github.com/99designs/gqlgen/graphql/introspection.Schema"},
		"Int":          {Model: "github.com/99designs/gqlgen/graphql.Int"},
		"Float":        {Model: "github.com/99designs/gqlgen/graphql.Float"},
		"String":       {Model: "github.com/99designs/gqlgen/graphql.String"},
		"Boolean":      {Model: "github.com/99designs/gqlgen/graphql.Boolean"},
		"ID":           {Model: "github.com/99designs/gqlgen/graphql.ID"},
		"Time":         {Model: "github.com/99designs/gqlgen/graphql.Time"},
		"Map":          {Model: "github.com/99designs/gqlgen/graphql.Map"},
	}

	if cfg.Models == nil {
		cfg.Models = TypeMap{}
	}
	for typeName, entry := range builtins {
		if !cfg.Models.Exists(typeName) {
			cfg.Models[typeName] = entry
		}
	}

	return nil
}

func (c *Config) LoadSchema() (*ast.Schema, map[string]string, error) {
	schemaStrings := map[string]string{}

	var sources []*ast.Source

	for _, filename := range c.SchemaFilename {
		var err error
		var schemaRaw []byte
		schemaRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
			os.Exit(1)
		}
		schemaStrings[filename] = string(schemaRaw)
		sources = append(sources, &ast.Source{Name: filename, Input: schemaStrings[filename]})
	}

	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return nil, nil, err
	}
	return schema, schemaStrings, nil
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func sanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}
