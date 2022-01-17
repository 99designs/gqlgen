package cmd

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func cleanupGenerate() {
	// remove generated files
	_ = os.Remove("gqlgen.yml")
	_ = os.Remove("server.go")
	_ = os.RemoveAll("graph")
}

func TestInitCmd(t *testing.T) {
	// setup test dir
	wd, _ := os.Getwd()
	testpath := path.Join(wd, "testdata", "init")
	defer func() {
		// remove generated files
		cleanupGenerate()
		_ = os.Chdir(wd)
	}()
	_ = os.Chdir(testpath)

	// Should ok if dir is empty
	app := cli.NewApp()
	app.Commands = []*cli.Command{initCmd}
	args := os.Args[0:1]
	args = append(args, "init")
	err := app.Run(args)
	require.Nil(t, err)

	// Should fail if dir is not empty, e.g. gqlgen.yml exists
	err = app.Run(args)
	require.NotNil(t, err)
	require.Equal(t, "gqlgen.yml already exists", err.Error())
}
