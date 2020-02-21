package cmd

import (
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var genCmd = &cli.Command{
	Name:  "generate",
	Usage: "generate a graphql server based on schema",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		&cli.StringFlag{Name: "config, c", Usage: "the config filename"},
	},
	Action: func(ctx *cli.Context) error {
		var cfg *config.Config
		var err error
		if configFilename := ctx.String("config"); configFilename != "" {
			cfg, err = config.LoadConfig(configFilename)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
				return err
			}
		} else {
			cfg, err = config.LoadConfigFromDefaultLocations()
			if os.IsNotExist(errors.Cause(err)) {
				cfg = config.DefaultConfig()
			} else if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(2)
				return err
			}
		}

		if err = api.Generate(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(3)
			return err
		}
		return nil
	},
}
