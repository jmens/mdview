//go:build windows

package main

import "os/exec"

// openCommand uses rundll32 with the URL.dll FileProtocolHandler entry point.
// This avoids spawning a visible cmd.exe window (as `cmd /c start` would) and
// passes the URL as a single argument, so no shell-quoting surprises.
func openCommand(url string) *exec.Cmd {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
}
