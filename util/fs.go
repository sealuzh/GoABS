package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	vendorFolder    = "vendor"
	usVendorFolder  = "_vendor"
	workspaceFolder = "_workspace"
)

func ExpandTilde(path string) (string, error) {
	home := ""

	switch runtime.GOOS {
	case "windows":
		home = filepath.Join(os.Getenv("HomeDrive"), os.Getenv("HomePath"))
		if home == "" {
			home = os.Getenv("UserProfile")
		}

	default:
		home = os.Getenv("HOME")
	}

	if home == "" {
		return "", fmt.Errorf("No home directory set")
	}

	return strings.Replace(path, "~", home, -1), nil
}

func IsValidDir(path string) bool {
	pathElems := strings.Split(path, string(filepath.Separator))

	for _, el := range pathElems {
		// remove everything from dependencies folder
		if el == vendorFolder || el == usVendorFolder || el == workspaceFolder {
			return false
		}
		// remove all hidden folders
		if strings.HasPrefix(el, ".") {
			return false
		}
	}

	return true
}
