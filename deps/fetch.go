package deps

import (
	"fmt"
	"os"

	"bitbucket.org/sealuzh/goptc/executil"
)

func Fetch(projectPath string) error {
	err := os.Chdir(projectPath)
	if err != nil {
		return err
	}
	depMgr := depMgr(projectPath)
	env := executil.Env(executil.GoPath(projectPath))
	out, err := depMgr.FetchDeps(env)
	if err != nil {
		return fmt.Errorf("Error while fetching dependencies for '%s': %v\n\nOut: %s", projectPath, err, string(out))
	}
	return nil
}
