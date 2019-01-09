package cmd

import (
	"fmt"

	"github.com/monzo/gqlgen/graphql"
	"github.com/urfave/cli"
)

var versionCmd = cli.Command{
	Name:  "version",
	Usage: "print the version string",
	Action: func(ctx *cli.Context) {
		fmt.Println(graphql.Version)
	},
}
