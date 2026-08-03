package router

import "os/exec"

// Reload sends nginx -s reload.
func Reload() error {
	return exec.Command("nginx", "-s", "reload").Run()
}