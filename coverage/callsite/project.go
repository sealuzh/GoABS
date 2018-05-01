package callsite

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sealuzh/goabs/utils/fsutil"
)

const (
	defaultFiles    = 10
	defaultPackages = 10
)

func PackageFromFile(path string) string {
	if strings.HasSuffix(path, ".go") {
		return path[:strings.LastIndex(path, "/")]
	}
	return path
}

func Files(path string, excludeTests bool) []string {
	if strings.HasSuffix(path, ".go") {
		return []string{path}
	}
	files := make([]string, 0, defaultFiles)
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		if !fsutil.IsValidDir(path) {
			return filepath.SkipDir
		}

		// skip non-go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if excludeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		files = append(files, path)

		return nil
	})
	return files
}

func Packages(path string, gopath string, recursivePackages bool) []string {
	pkg := PackageFromFile(path)
	startPkg := filepath.Join(gopath, "src", pkg)

	if recursivePackages {
		ret := make([]string, 0, defaultPackages)
		filepath.Walk(startPkg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return nil
			}

			if !fsutil.IsValidDir(path) {
				return filepath.SkipDir
			}

			ret = append(ret, path[strings.Index(path, pkg):])
			return nil
		})
		return ret
	}

	// only file -> consider just the package in the file
	return []string{pkg}
}
