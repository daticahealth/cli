package git

import "os/exec"

// Initialize a new git repo in the current directory
func (g *SGit) Create() error {
	_, err := exec.Command("git", "init").Output()
	return err
}
