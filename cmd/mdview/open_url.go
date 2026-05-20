package main

import (
	"fmt"
	"os"
)

// openExternal launches the OS default handler for url and does not wait.
// The per-platform openCommand decides which executable to invoke.
func openExternal(url string) {
	cmd := openCommand(url)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "mdview: open external failed:", err)
		return
	}
	go func() { _ = cmd.Wait() }()
}
