package deps

import (
	"os"
	"path/filepath"
)

const (
	get       DepMgr = "Get"
	glide     DepMgr = "Glide"
	godep     DepMgr = "Godep"
	govendor  DepMgr = "Govendor"
	submodule DepMgr = "manul"
	gvt       DepMgr = "gvt"
	govend    DepMgr = "govend"
	trash     DepMgr = "trash"
	gom       DepMgr = "gom"
	gopm      DepMgr = "gopm"
	gogradle  DepMgr = "Gogradle"
	gpm       DepMgr = "gpm"
	glock     DepMgr = "glock"
)

var goGet = []string{"go", "get", "$(go list ./... | grep -v -E 'vendor|_vendor|.vendor|_workspace')"}

type DepMgr string

func (d DepMgr) String() string {
	return string(d)
}

func (d DepMgr) InstallCmd() []string {
	var cmd []string
	switch d {
	case glide:
		cmd = []string{"glide", "install"}
	case godep:
		cmd = []string{"godep", "restore"}
	case govendor:
		cmd = []string{"govendor", "sync"}
	case submodule:
		cmd = []string{"manul", "-I"}
	case gvt:
		cmd = []string{"gvt", "fetch"}
	case govend:
		cmd = []string{"govend", "-v"}
	case trash:
		cmd = []string{"trash"}
	case gom:
		cmd = []string{"gom", "install"}
	case gopm:
		cmd = []string{"gopm", "get"}
	case gogradle:
		cmd = []string{"./gradlew", "vendor"}
	case gpm:
		cmd = []string{"gpm", "install"}
	case glock:
		cmd = []string{"glock", "sync"}

	case get:
		fallthrough
	default:
		cmd = goGet
	}
	return cmd
}

// based on https://github.com/blindpirate/report-of-build-tools-for-java-and-golang and
// https://github.com/golang/go/wiki/PackageManagementTools
func depMgr(projectPath string) DepMgr {
	// Godeps
	p := filepath.Join(projectPath, "Godeps/Godeps.json")
	_, err := os.Stat(p)
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
	p = filepath.Join(projectPath, ".gitsubmodules")
	_, err = os.Stat(p)
	if err == nil {
		return submodule
	}

	return get
}
