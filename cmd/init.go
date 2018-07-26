package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlgen/codegen"
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
	Short: "Generate .gqlgen.yml",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		initSchema()
		initConfig()
	},
}

func initConfig() {
	var config *codegen.Config
	var err error
	if configFilename != "" {
		config, err = codegen.LoadConfig(configFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		} else if config != nil {
			fmt.Fprintln(os.Stderr, "config file is already exists")
			os.Exit(0)
		}
	} else {
		config, err = codegen.LoadConfigFromDefaultLocations()
		if os.IsNotExist(errors.Cause(err)) {
			if configFilename == "" {
				configFilename = ".gqlgen.yml"
			}
			config = codegen.DefaultConfig()
		} else if config != nil {
			fmt.Fprintln(os.Stderr, "config file is already exists")
			os.Exit(0)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
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
