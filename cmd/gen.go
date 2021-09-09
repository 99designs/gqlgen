package cmd

import (
	"errors"
	"io/fs"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
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
				return err
			}
		} else {
			cfg, err = config.LoadConfigFromDefaultLocations()
			if errors.Is(err, fs.ErrNotExist) {
				cfg, err = config.LoadDefaultConfig()
			}

			if err != nil {
				return err
			}
		}

		if err = api.Generate(cfg); err != nil {
			return err
		}
		return nil
	},
}
