package executil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	srcFolder      = "src"
	goPathVariable = "GOPATH"
	goRootVariable = "GOROOT"
)

func Env(goRoot, goPath string) []string {
	env := os.Environ()
	ret := make([]string, 0, len(env)+1)
	goRootDecl := fmt.Sprintf("%s=%s", goRootVariable, goRoot)
	goPathDecl := fmt.Sprintf("%s=%s", goPathVariable, goPath)
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, goRootVariable) && goRoot != "":
			ret = append(ret, goRootDecl)
		case strings.HasPrefix(e, goPathVariable) && goPath != "":
			ret = append(ret, goPathDecl)
		default:
			ret = append(ret, e)
		}
	}
	return ret
}

func GoPath(p string) string {
	pathArr := strings.Split(p, string(filepath.Separator))
	var c int
	for i, el := range pathArr {
		if el == srcFolder {
			c = i
			break
		}
	}
	return fmt.Sprintf("/%s", filepath.Join(pathArr[:c]...))
}
