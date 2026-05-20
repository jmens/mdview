//go:build darwin

package main

import "os/exec"

func openCommand(url string) *exec.Cmd {
	return exec.Command("open", url)
}
