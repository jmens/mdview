//go:build linux

package main

import "testing"

func TestOpenCommandLinux(t *testing.T) {
	cmd := openCommand("https://example.com")
	if cmd.Args[0] != "xdg-open" {
		t.Fatalf("expected xdg-open, got %q", cmd.Args[0])
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "https://example.com" {
		t.Fatalf("expected args [xdg-open https://example.com], got %v", cmd.Args)
	}
}
