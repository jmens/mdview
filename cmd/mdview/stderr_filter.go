//go:build linux

package main

import (
	"bufio"
	"bytes"
	"os"
	"syscall"
)

// WebKit, GTK, and JavaScriptCore emit harmless noise to stderr on startup
// and shutdown (signal-handler overrides, the JSC_SIGNAL_FOR_GC hint,
// option-parser warnings). Filter the specific lines we know about and
// pass everything else through unchanged.
var stderrSuppress = [][]byte{
	[]byte("Overriding existing handler for signal"),
	[]byte("Set JSC_SIGNAL_FOR_GC"),
	[]byte("invalid option: JSC_"),
}

// startStderrFilter redirects fd 2 through a pipe and drops lines that match
// known WebKit noise patterns. The returned function restores the original
// stderr and stops the reader goroutine.
func startStderrFilter() (func(), error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	origFd, err := syscall.Dup(2)
	if err != nil {
		_ = r.Close()
		_ = w.Close()
		return nil, err
	}
	origStderr := os.NewFile(uintptr(origFd), "/dev/stderr.orig")

	if err := syscall.Dup2(int(w.Fd()), 2); err != nil {
		_ = r.Close()
		_ = w.Close()
		_ = origStderr.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 16*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Bytes()
			if isNoise(line) {
				continue
			}
			_, _ = origStderr.Write(line)
			_, _ = origStderr.Write([]byte{'\n'})
		}
	}()

	cleanup := func() {
		_ = syscall.Dup2(origFd, 2)
		_ = w.Close()
		<-done
		_ = r.Close()
		_ = origStderr.Close()
	}
	return cleanup, nil
}

func isNoise(line []byte) bool {
	for _, p := range stderrSuppress {
		if bytes.Contains(line, p) {
			return true
		}
	}
	return false
}
