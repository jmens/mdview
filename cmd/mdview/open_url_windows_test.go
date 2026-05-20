//go:build windows

package main

import "testing"

func TestOpenCommandWindows(t *testing.T) {
	cmd := openCommand("https://example.com")
	if cmd.Args[0] != "rundll32.exe" {
		t.Fatalf("expected rundll32.exe, got %q", cmd.Args[0])
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "url.dll,FileProtocolHandler" || cmd.Args[2] != "https://example.com" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
