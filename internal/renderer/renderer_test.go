package renderer

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestRender_Goldens(t *testing.T) {
	r := New()
	entries, err := filepath.Glob(filepath.Join("testdata", "*.md"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no testdata/*.md fixtures found")
	}
	for _, in := range entries {
		name := strings.TrimSuffix(filepath.Base(in), ".md")
		t.Run(name, func(t *testing.T) {
			res, err := r.RenderFile(in)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			golden := filepath.Join("testdata", name+".html.golden")
			got := res.HTML
			if *update {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden: %v (run with -update to create)", err)
			}
			if got != string(want) {
				t.Errorf("mismatch (re-run with -update if intended)\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
			}
		})
	}
}

func TestRender_TitleFromH1(t *testing.T) {
	r := New()
	res, err := r.Render([]byte("# Hello World\n\nbody\n"), "fallback.md")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if res.Title != "Hello World" {
		t.Errorf("title = %q, want %q", res.Title, "Hello World")
	}
}

func TestRender_TitleFallback(t *testing.T) {
	r := New()
	res, err := r.Render([]byte("no heading here\n"), "fallback.md")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if res.Title != "fallback.md" {
		t.Errorf("title = %q, want fallback", res.Title)
	}
}
