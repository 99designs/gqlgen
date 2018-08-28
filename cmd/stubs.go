package cmd

import (
	"fmt"

	"github.com/99designs/gqlgen/codegen"
	"github.com/urfave/cli"
)

var stubsCommand = cli.Command{
	Name:  "stubs",
	Usage: "generate empty resolver stubs to match the schema",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "verbose, v", Usage: "show logs"},
		cli.BoolFlag{Name: "force, f", Usage: "throw caution to the wind"},
		cli.StringFlag{Name: "config, c", Usage: "the config filename"},
	},
	Action: func(ctx *cli.Context) error {
		if !ctx.Bool("force") {
			fmt.Printf("WARNING: This feature is experimental, it might trash your code. Run again with -f to do it anyway.\n")
			return fmt.Errorf("aborted")
		}

		cfg, err := loadConfig(ctx)
		if err != nil {
			return err
		}

		return codegen.Stubs(cfg)
	},
}
