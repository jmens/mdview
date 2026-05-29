package doctype

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		path string
		want Type
	}{
		{"foo.md", Markdown},
		{"foo.markdown", Markdown},
		{"FOO.MD", Markdown},
		{"foo.pdf", PDF},
		{"foo.PDF", PDF},
		{"path/to/document.pdf", PDF},
		{"no-extension", Markdown},
		{"archive.tar.gz", Markdown},
	}
	for _, c := range cases {
		if got := Detect(c.path); got != c.want {
			t.Errorf("Detect(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestType_String(t *testing.T) {
	if Markdown.String() != "markdown" {
		t.Errorf("Markdown.String() = %q, want %q", Markdown.String(), "markdown")
	}
	if PDF.String() != "pdf" {
		t.Errorf("PDF.String() = %q, want %q", PDF.String(), "pdf")
	}
}
