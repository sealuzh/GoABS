package deps

import (
	"fmt"
	"os/exec"
)

func Fetch(projectPath string) error {
	depMgr := depMgr(projectPath)
	cmdArr := depMgr.InstallCmd()
	c := exec.Command(cmdArr[0])
	if len(cmdArr) > 0 {
		c.Args = cmdArr[1:]
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error while fetching dependencies for '%s': %v\n\nOut: %s", projectPath, err, string(out))
	}
	return nil
}
