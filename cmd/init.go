package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/internal/code"
	"github.com/99designs/gqlgen/plugin/servergen"
	"github.com/urfave/cli/v2"
)

var configTemplate = template.Must(template.New("name").Parse(
	`# Where are all the schema files located? globs are supported eg  src/**/*.graphqls
schema:
  - graph/*.graphqls

# Where should the generated server code go?
exec:
  filename: graph/generated/generated.go
  package: generated

# Uncomment to enable federation
# federation:
#   filename: graph/generated/federation.go
#   package: generated

# Where should any generated models go?
model:
  filename: graph/model/models_gen.go
  package: model

# Where should the resolver implementations go?
resolver:
  layout: follow-schema
  dir: graph
  package: graph

# Optional: turn on use ` + "`" + `gqlgen:"fieldName"` + "`" + ` tags in your models
# struct_tag: json

# Optional: turn on to use []Thing instead of []*Thing
# omit_slice_element_pointers: false

# Optional: set to speed up generation time by not performing a final validation pass.
# skip_validation: true

# gqlgen will search for any type names in the schema in these go packages
# if they match it will use them, otherwise it will generate them.
autobind:
  - "{{.}}/graph/model"

# This section declares type mapping between the GraphQL and go type systems
#
# The first line in each type will be used as defaults for resolver arguments and
# modelgen, the others will be allowed when binding to fields. Configure them to
# your liking
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
  Int:
    model:
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
`))

var schemaDefault = `# GraphQL schema example
#
# https://gqlgen.com/getting-started/

type Todo {
  id: ID!
  text: String!
  done: Boolean!
  user: User!
}

type User {
  id: ID!
  name: String!
}

type Query {
  todos: [Todo!]!
}

input NewTodo {
  text: String!
  userId: String!
}

type Mutation {
  createTodo(input: NewTodo!): Todo!
}
`

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "create a new gqlgen project",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		&cli.StringFlag{Name: "config, c", Usage: "the config filename"},
		&cli.StringFlag{Name: "server", Usage: "where to write the server stub to", Value: "server.go"},
		&cli.StringFlag{Name: "schema", Usage: "where to write the schema stub to", Value: "graph/schema.graphqls"},
	},
	Action: func(ctx *cli.Context) error {
		configFilename := ctx.String("config")
		serverFilename := ctx.String("server")

		pkgName := code.ImportPathForDir(".")
		if pkgName == "" {
			return fmt.Errorf("unable to determine import path for current directory, you probably need to run go mod init first")
		}

		if err := initFile(ctx.String("schema"), schemaDefault); err != nil {
			return err
		}

		cfg, err := loadConfig(configFilename)

		if err != nil {
			if configFilename == "" {
				configFilename = "gqlgen.yml"
			}

			fmt.Println("Creating", configFilename)
			if err := initFile(configFilename, executeConfigTemplate(pkgName)); err != nil {
				return err
			}

			// create the package directory with a temporary file so that go recognises it as a package
			// and autobinding doesn't error out
			tmpPackageNameFile := "graph/model/_tmp_gqlgen_init.go"
			if err := initFile(tmpPackageNameFile, "package model"); err != nil {
				return err
			}
			defer os.Remove(tmpPackageNameFile)

			if cfg, err = loadConfig(configFilename); err != nil {
				panic(err)
			}
		} else {
			fmt.Println("Skipping creating gqlgen.yml as it already exists")
		}

		fmt.Println("Creating graph/...")
		fmt.Println("Creating", serverFilename)
		if err := api.Generate(cfg, api.AddPlugin(servergen.New(serverFilename))); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}

		fmt.Fprintf(os.Stdout, "\nExec \"go run ./%s\" to start GraphQL server\n", serverFilename)
		return nil
	},
}

func loadConfig(configFilename string) (*config.Config, error) {
	if configFilename != "" {
		return config.LoadConfig(configFilename)
	} else {
		return config.LoadConfigFromDefaultLocations()
	}
}

func executeConfigTemplate(pkgName string) string {
	var buf bytes.Buffer
	if err := configTemplate.Execute(&buf, pkgName); err != nil {
		panic(err)
	}

	return buf.String()
}

func initFile(filename, contents string) error {
	_, err := os.Stat(filename)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return fmt.Errorf("unable to create directory for file '%s': %w", filename, err)
		}
		if err = ioutil.WriteFile(filename, []byte(contents), 0644); err != nil {
			return fmt.Errorf("unable to write file '%s': %w", filename, err)
		}
	}
	return nil
}
