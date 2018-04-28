package codegen

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/codegen/templates"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/imports"
)

type Config struct {
	SchemaStr string
	Typemap   map[string]string

	schema *schema.Schema

	ExecFilename    string
	ExecPackageName string
	execPackagePath string
	execDir         string

	ModelFilename    string
	ModelPackageName string
	modelPackagePath string
	modelDir         string
}

func Generate(cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}

	_ = syscall.Unlink(cfg.ExecFilename)
	_ = syscall.Unlink(cfg.ModelFilename)

	modelsBuild, err := cfg.models()
	if err != nil {
		return errors.Wrap(err, "model plan failed")
	}
	if len(modelsBuild.Models) > 0 {
		modelsBuild.PackageName = cfg.ModelPackageName
		var buf *bytes.Buffer
		buf, err = templates.Run("models.gotpl", modelsBuild)
		if err != nil {
			return errors.Wrap(err, "model generation failed")
		}

		if err = write(cfg.ModelFilename, buf.Bytes()); err != nil {
			return err
		}
		for _, model := range modelsBuild.Models {
			cfg.Typemap[model.GQLType] = cfg.modelPackagePath + "." + model.GoType
		}

		for _, enum := range modelsBuild.Enums {
			cfg.Typemap[enum.GQLType] = cfg.modelPackagePath + "." + enum.GoType
		}
	}

	build, err := cfg.bind()
	if err != nil {
		return errors.Wrap(err, "exec plan failed")
	}
	build.SchemaRaw = cfg.SchemaStr
	build.PackageName = cfg.ExecPackageName

	var buf *bytes.Buffer
	buf, err = templates.Run("generated.gotpl", build)
	if err != nil {
		return errors.Wrap(err, "exec codegen failed")
	}

	if err = write(cfg.ExecFilename, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) normalize() error {
	if cfg.ModelFilename == "" {
		return errors.New("ModelFilename is required")
	}
	cfg.ModelFilename = abs(cfg.ModelFilename)
	cfg.modelDir = filepath.Dir(cfg.ModelFilename)
	if cfg.ModelPackageName == "" {
		cfg.ModelPackageName = filepath.Base(cfg.modelDir)
	}
	cfg.ModelPackageName = sanitizePackageName(cfg.ModelPackageName)
	cfg.modelPackagePath = fullPackageName(cfg.modelDir, cfg.ModelPackageName)

	if cfg.ExecFilename == "" {
		return errors.New("ModelFilename is required")
	}
	cfg.ExecFilename = abs(cfg.ExecFilename)
	cfg.execDir = filepath.Dir(cfg.ExecFilename)
	if cfg.ExecPackageName == "" {
		cfg.ExecPackageName = filepath.Base(cfg.execDir)
	}
	cfg.ExecPackageName = sanitizePackageName(cfg.ExecPackageName)
	cfg.execPackagePath = fullPackageName(cfg.execDir, cfg.ExecPackageName)

	builtins := map[string]string{
		"__Directive":  "github.com/vektah/gqlgen/neelance/introspection.Directive",
		"__Type":       "github.com/vektah/gqlgen/neelance/introspection.Type",
		"__Field":      "github.com/vektah/gqlgen/neelance/introspection.Field",
		"__EnumValue":  "github.com/vektah/gqlgen/neelance/introspection.EnumValue",
		"__InputValue": "github.com/vektah/gqlgen/neelance/introspection.InputValue",
		"__Schema":     "github.com/vektah/gqlgen/neelance/introspection.Schema",
		"Int":          "github.com/vektah/gqlgen/graphql.Int",
		"Float":        "github.com/vektah/gqlgen/graphql.Float",
		"String":       "github.com/vektah/gqlgen/graphql.String",
		"Boolean":      "github.com/vektah/gqlgen/graphql.Boolean",
		"ID":           "github.com/vektah/gqlgen/graphql.ID",
		"Time":         "github.com/vektah/gqlgen/graphql.Time",
		"Map":          "github.com/vektah/gqlgen/graphql.Map",
	}

	if cfg.Typemap == nil {
		cfg.Typemap = map[string]string{}
	}
	for k, v := range builtins {
		if _, ok := cfg.Typemap[k]; !ok {
			cfg.Typemap[k] = v
		}
	}

	cfg.schema = schema.New()
	return cfg.schema.Parse(cfg.SchemaStr)
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return absPath
}

func fullPackageName(dir string, pkgName string) string {
	fullPkgName := filepath.Join(filepath.Dir(dir), pkgName)

	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		gopath = filepath.Join(gopath, "src") + string(os.PathSeparator)
		fullPkgName = strings.TrimPrefix(fullPkgName, gopath)
	}
	return filepath.ToSlash(fullPkgName)
}

func gofmt(filename string, b []byte) ([]byte, error) {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		return b, errors.Wrap(err, "unable to gofmt")
	}
	return out, nil
}

func write(filename string, b []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	formatted, err := gofmt(filename, b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt failed: %s\n", err.Error())
		formatted = b
	}

	err = ioutil.WriteFile(filename, formatted, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", filename)
	}

	return nil
}
