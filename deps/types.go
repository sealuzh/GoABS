package deps

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sealuzh/goabs/utils/executil"
)

const (
	get      DepMgr = "Get"
	dep      DepMgr = "dep"
	glide    DepMgr = "Glide"
	godep    DepMgr = "Godep"
	govendor DepMgr = "Govendor"
	//submodule DepMgr = "manul"
	gvt      DepMgr = "gvt"
	govend   DepMgr = "govend"
	trash    DepMgr = "trash"
	gom      DepMgr = "gom"
	gopm     DepMgr = "gopm"
	gogradle DepMgr = "Gogradle"
	gpm      DepMgr = "gpm"
	glock    DepMgr = "glock"
)

const (
	shellCmd     = "/bin/bash"
	shellCmdCArg = "-c"
)

const (
	goCmd        = "go"
	goGet        = "get"
	goList       = "list"
	goAllPkgs    = "./..."
	goListNoDeps = "go list ./... | grep -v -E 'vendor|_vendor|.vendor|_workspace'"
)

var depFolders = []string{"vendor", "_vendor", ".vendor", "_workspace"}

type DepMgr string

func (d DepMgr) String() string {
	return string(d)
}

func (d DepMgr) FetchDeps(env []string) ([]byte, error) {
	if d == get {
		return execGoGet(env)
	}

	var c *exec.Cmd
	cmdArr := strings.Split(d.installCmd(), " ")
	cmdName := cmdArr[0]

	c = exec.Command(cmdName)
	if len(cmdArr) > 1 {
		c.Args = cmdArr
	}

	if len(env) > 0 {
		c.Env = env
	}

	return c.CombinedOutput()
}

func execGoGet(env []string) ([]byte, error) {
	goCommand := executil.GoCommand(env)

	setEnv := len(env) > 0
	c := exec.Command(goCommand, goList, goAllPkgs)
	if setEnv {
		c.Env = env
	}

	out, err := c.CombinedOutput()
	if err != nil {
		return out, err
	}

	r := bufio.NewReader(bytes.NewBuffer(out))
	var outBuf bytes.Buffer
Loop:
	for {
		pkg, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break Loop
			}
			return outBuf.Bytes(), err
		}

		pkg = strings.TrimSpace(pkg)
		if depsFolderInPath(pkg) {
			continue Loop
		}

		cmd := exec.Command(goCommand, goGet, pkg)
		if setEnv {
			cmd.Env = env
		}
		out, err = cmd.CombinedOutput()
		outBuf.Write(out)
		if err != nil {
			return outBuf.Bytes(), err
		}
	}
	return outBuf.Bytes(), nil
}

func depsFolderInPath(path string) bool {
	goPath := executil.GoPath(path)
	for _, f := range depFolders {
		if strings.Contains(goPath, f) {
			return true
		}
	}
	return false
}

func (d DepMgr) installCmd() string {
	var cmd string
	switch d {
	case dep:
		cmd = "dep ensure"
	case glide:
		cmd = "glide install"
	case godep:
		cmd = "godep restore"
	case govendor:
		cmd = "govendor sync"
	//case submodule:
	//	cmd = "manul -I"
	case gvt:
		cmd = "gvt restore"
	case govend:
		cmd = "govend -v"
	case trash:
		cmd = "trash"
	case gom:
		cmd = "gom install"
	case gopm:
		cmd = "gopm get"
	case gogradle:
		cmd = "./gradlew vendor"
	case gpm:
		cmd = "gpm install"
	case glock:
		cmd = "glock sync"

	case get:
		fallthrough
	default:
		cmd = fmt.Sprintf("%s %s %s", goCmd, goGet, goAllPkgs)
	}
	return cmd
}

// based on https://github.com/blindpirate/report-of-build-tools-for-java-and-golang and
// https://github.com/golang/go/wiki/PackageManagementTools
func Manager(projectPath string) DepMgr {
	// dep
	p := filepath.Join(projectPath, "Gopkg.lock")
	_, err := os.Stat(p)
	if err == nil {
		return dep
	}

	// Godeps
	p = filepath.Join(projectPath, "Godeps/Godeps.json")
	_, err = os.Stat(p)
	if err == nil {
		return godep
	}

	// govendor
	p = filepath.Join(projectPath, "vendor/vendor.json")
	_, err = os.Stat(p)
	if err == nil {
		return govendor
	}

	// gopm
	p = filepath.Join(projectPath, ".gopmfile")
	_, err = os.Stat(p)
	if err == nil {
		return gopm
	}

	// gvt
	p = filepath.Join(projectPath, "vendor/manifest")
	_, err = os.Stat(p)
	if err == nil {
		return gvt
	}

	// govend
	p = filepath.Join(projectPath, "vendor.yml")
	_, err = os.Stat(p)
	if err == nil {
		return govend
	}

	// Glide
	p = filepath.Join(projectPath, "glide.yaml")
	_, err = os.Stat(p)
	if err == nil {
		return glide
	}
	p = filepath.Join(projectPath, "glide.lock")
	_, err = os.Stat(p)
	if err == nil {
		return glide
	}

	// trash
	p = filepath.Join(projectPath, "vendor.conf")
	_, err = os.Stat(p)
	if err == nil {
		return trash
	}
	p = filepath.Join(projectPath, "glide.yml")
	_, err = os.Stat(p)
	if err == nil {
		return trash
	}
	p = filepath.Join(projectPath, "trash.yaml")
	_, err = os.Stat(p)
	if err == nil {
		return trash
	}

	// gom
	p = filepath.Join(projectPath, "Gomfile")
	_, err = os.Stat(p)
	if err == nil {
		return gom
	}

	// gogradle
	p = filepath.Join(projectPath, "gradlew")
	_, err = os.Stat(p)
	if err == nil {
		return gogradle
	}

	// gpm
	p = filepath.Join(projectPath, "Godeps")
	fi, err := os.Stat(p)
	if err == nil && !fi.IsDir() {
		return gpm
	}

	// glock
	p = filepath.Join(projectPath, "GLOCKFILE")
	_, err = os.Stat(p)
	if err == nil {
		return glock
	}

	// submodule
	// p = filepath.Join(projectPath, ".gitsubmodules")
	// _, err = os.Stat(p)
	// if err == nil {
	// 	return submodule
	// }

	return get
}
