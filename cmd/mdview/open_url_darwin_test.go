//go:build darwin

package main

import "testing"

func TestOpenCommandDarwin(t *testing.T) {
	cmd := openCommand("https://example.com")
	if cmd.Args[0] != "open" {
		t.Fatalf("expected open, got %q", cmd.Args[0])
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "https://example.com" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
