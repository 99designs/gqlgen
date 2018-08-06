package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var configComment = `
# .gqlgen.yml example
#
# Refer to https://gqlgen.com/config/
# for detailed .gqlgen.yml documentation.
`

var schemaDefault = `
# GraphQL schema example
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate gqlgen skeleton",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		initSchema()
		config := initConfig()

		GenerateGraphServer(config)
	},
}

func GenerateGraphServer(config *codegen.Config) {
	schemaRaw, err := ioutil.ReadFile(config.SchemaFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
		os.Exit(1)
	}
	config.SchemaStr = string(schemaRaw)

	if err = config.Check(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config format: "+err.Error())
		os.Exit(1)
	}

	if serverFilename == "" {
		serverFilename = "server/server.go"
	}

	if err := codegen.Generate(*config); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := codegen.GenerateServer(*config, serverFilename); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, `Exec "go run ./%s" to start GraphQL server`, serverFilename)
}

func initConfig() *codegen.Config {
	var config *codegen.Config
	var err error
	if configFilename != "" {
		config, err = codegen.LoadConfig(configFilename)
	} else {
		config, err = codegen.LoadConfigFromDefaultLocations()
	}

	if config != nil {
		fmt.Fprintf(os.Stderr, "init failed: a configuration file already exists at %s\n", config.FilePath)
		os.Exit(1)
	}

	if !os.IsNotExist(errors.Cause(err)) {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if configFilename == "" {
		configFilename = "gqlgen.yml"
	}
	config = codegen.DefaultConfig()

	config.Resolver = codegen.PackageConfig{
		Filename: "resolver.go",
		Type:     "Resolver",
	}

	if schemaFilename != "" {
		config.SchemaFilename = schemaFilename
	}
	if models != "" {
		config.Model.Filename = models
	}
	if output != "" {
		config.Exec.Filename = output
	}
	if packageName != "" {
		config.Exec.Package = packageName
	}
	if modelPackageName != "" {
		config.Model.Package = modelPackageName
	}
	if typemap != "" {
		config.Models = loadModelMap()
	}

	var buf bytes.Buffer
	buf.WriteString(strings.TrimSpace(configComment))
	buf.WriteString("\n\n")
	{
		var b []byte
		b, err = yaml.Marshal(config)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to marshal yaml: "+err.Error())
			os.Exit(1)
		}
		buf.Write(b)
	}

	err = ioutil.WriteFile(configFilename, buf.Bytes(), 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to write config file: "+err.Error())
		os.Exit(1)
	}

	return config
}

func initSchema() {
	if schemaFilename == "" {
		schemaFilename = "schema.graphql"
	}

	_, err := os.Stat(schemaFilename)
	if !os.IsNotExist(err) {
		return
	}

	err = ioutil.WriteFile(schemaFilename, []byte(strings.TrimSpace(schemaDefault)), 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to write schema file: "+err.Error())
		os.Exit(1)
	}
}
