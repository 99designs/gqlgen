package gopath

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var NotFound = fmt.Errorf("not on GOPATH")
var ModuleNameRegexp = regexp.MustCompile(`module\s+(.*)`)

// Contains returns true if the given directory is in the GOPATH
func Contains(dir string) bool {
	_, err := Dir2Import(dir)
	return err == nil
}

// Dir2Import takes an *absolute* path and returns a golang import path for the package, and returns an error if it isn't on the gopath
func Dir2Import(dir string) (string, error) {
	dir = filepath.ToSlash(dir)
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		gopath = filepath.ToSlash(filepath.Join(gopath, "src"))
		if len(gopath) < len(dir) && strings.EqualFold(gopath, dir[0:len(gopath)]) {
			return dir[len(gopath)+1:], nil
		}
	}

	// The following code handles the go modules in a manner that does not break existing code. However, it is not
	// efficient because it requires several round trips to the filesystem. It also should technically be the first
	// method we try (before GOPATH) because if someone creates a 'go mod' project that also happens to be in the
	// GOPATH, the import path will be calculated from GOPATH and not from the go.mod file.
	//
	// Possible Fixes:
	// 1) Cache the results of scanning the filesystem. The code is typically called at compile-time, so its unlikely
	//    for a go.mod file to be added or altered during compile.
	// 2) Add an optional switch that turns off GOPATH checks entirely when go.mod is used on a project.
	// 3) Rewrite gqlgen to make use of 'golang.org/x/tools/go/packages' for determining import paths instead.

	// Scan the path tree for a directory that contains a 'go.mod' file and read the module name from it
	modDirectory := findGoMod(dir)
	if 0 == len(modDirectory) {
		return "", NotFound
	}
	modName := moduleName(filepath.Join(modDirectory, "go.mod"))
	if 0 == len(modName) {
		return "", NotFound
	}

	// At this point 'dir' looks something like '/root/path/to/some/dir', 'modDirectory' looks like '/root/path',
	// and 'modName' looks like 'grabhub.com/myname/vunderprojekt'. The correct import path is:
	// 'grabhub.com/myname/vunderprojekt/to/some/dir'
	return fmt.Sprintf("%s%s", modName, strings.TrimPrefix(dir, filepath.ToSlash(modDirectory))), nil
}

// MustDir2Import takes an *absolute* path and returns a golang import path for the package, and panics if it isn't on the gopath
func MustDir2Import(dir string) string {
	pkg, err := Dir2Import(dir)
	if err != nil {
		panic(err)
	}
	return pkg
}

// Returns the path to the first go.mod file in the parent tree starting with the specified directory. Returns "" if not found
func findGoMod(srcDir string) string {
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return ""
	}
	for {
		info, err := os.Stat(filepath.Join(abs, "go.mod"))
		if err == nil && !info.IsDir() {
			break
		}
		d := filepath.Dir(abs)
		if len(d) >= len(abs) {
			return "" // reached top of file system, no go.mod
		}
		abs = d
	}

	return abs
}

// Returns the main module name from a go.mod file. Returns "" if it cannot be found
func moduleName(file string) string {
	data, err := ioutil.ReadFile(file)
	if nil != err {
		return ""
	}

	// Search for `module some/name`
	matches := ModuleNameRegexp.FindSubmatch(data)
	if len(matches) < 2 {
		return ""
	}

	// The first element is the whole line, the second is the capture group we specified for the module name.
	return string(matches[1])
}
