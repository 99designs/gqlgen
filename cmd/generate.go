package cmd

import (
	"io/ioutil"
	"os"

	"github.com/99designs/gqlgen/codegen"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var genCmd = cli.Command{
	Name:  "generate",
	Usage: "generate a graphql server based on schema",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		cli.StringFlag{Name: "config, c", Usage: "the config filename"},
	},
	Action: func(ctx *cli.Context) error {
		cfg, err := loadConfig(ctx)
		if err != nil {
			return err
		}

		return codegen.Generate(cfg)
	},
}

func loadConfig(ctx *cli.Context) (*codegen.Config, error) {
	var config *codegen.Config
	var err error

	if configFilename := ctx.String("config"); configFilename != "" {
		config, err = codegen.LoadConfig(configFilename)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = codegen.LoadConfigFromDefaultLocations()
		if os.IsNotExist(errors.Cause(err)) {
			config = codegen.DefaultConfig()
		} else if err != nil {
			return nil, err
		}
	}

	schemaRaw, err := ioutil.ReadFile(config.SchemaFilename)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open schema")
	}
	config.SchemaStr = string(schemaRaw)

	if err = config.Check(); err != nil {
		return nil, errors.Wrap(err, "invalid config format")
	}

	if err := config.Normalize(); err != nil {
		return nil, err
	}

	return config, nil
}
