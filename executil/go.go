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
)

func Env(goPath string) []string {
	env := os.Environ()
	ret := make([]string, 0, len(env)+1)
	added := false
	goPathDecl := fmt.Sprintf("%s=%s", goPathVariable, goPath)
	for _, e := range env {
		if strings.HasPrefix(e, goPathVariable) {
			ret = append(ret, goPathDecl)
			added = true
		} else {
			ret = append(ret, e)
		}
	}
	if !added {
		ret = append(ret, goPathDecl)
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
