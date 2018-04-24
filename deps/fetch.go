package deps

import (
	"fmt"
	"os"

	"github.com/sealuzh/goabs/util"
)

func Fetch(projectPath string) error {
	err := os.Chdir(projectPath)
	if err != nil {
		return err
	}
	depMgr := Manager(projectPath)
	gopath := util.GoPath(projectPath)
	env := util.Env(gopath)
	out, err := depMgr.FetchDeps(env)
	if err != nil {
		return fmt.Errorf("Error while fetching dependencies for '%s': %v\n\nOut: %s", projectPath, err, string(out))
	}
	return nil
}
