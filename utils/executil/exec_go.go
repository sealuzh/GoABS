package executil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	srcFolder      = "src"
	goCmd          = "go"
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
		case strings.HasPrefix(e, "PATH"):
			ret = append(ret, replacePath(e, goRoot))
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

func replacePath(path, goRoot string) string {
	if goRoot == "" {
		return path
	}

	return fmt.Sprintf(
		"PATH=%s/bin:%s",
		goRoot,
		path[5:],
	)
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

func GoCommand(env []string) string {
	cmd := goCmd
	for _, ev := range env {
		if strings.HasPrefix(ev, goRootVariable) {
			goRoot := ev[7:] // GOROOT=x
			cmd = fmt.Sprintf("%s/bin/%s", goRoot, goCmd)
		}
	}
	return cmd
}
