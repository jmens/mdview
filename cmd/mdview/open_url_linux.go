//go:build linux

package main

import "os/exec"

func openCommand(url string) *exec.Cmd {
	return exec.Command("xdg-open", url)
}
