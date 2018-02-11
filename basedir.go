package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var templateDir = func() string {
	var gopath = os.Getenv("GOPATH")
	if gopath == "" {
		usr, err := user.Current()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot determine home dir: %s\n", err.Error())
			os.Exit(1)
		}
		gopath = filepath.Join(usr.HomeDir, "go")
	}
	for _, path := range strings.Split(gopath, ":") {
		if path != "" {
			abspath, _ := filepath.Abs(filepath.Join(path, "src", "github.com", "vektah", "gqlgen"))
			if dirExists(abspath) {
				return abspath
			}
		}
	}

	fmt.Fprintln(os.Stderr, "cannot determine base of github.com/vektah/gqlgen")
	os.Exit(1)
	return ""
}()

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return !os.IsNotExist(err) && fi.IsDir()
}
