package deps

import "os/exec"

func Fetch(projectPath string) error {
	depMgr := depMgr(projectPath)
	c := exec.Command(depMgr.InstallCmd())
	err := c.Run()
	if err != nil {
		return err
	}
	return nil
}
