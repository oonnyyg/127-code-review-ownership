package main

import "testing"

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"a/b.go", "a/b.go", true},
		{"a/b.go", "a/c.go", false},
		{"a/*.go", "a/b.go", true},
		{"a/*.go", "a/x/b.go", false},
		{"*.go", "main.go", true},
		{"*.go", "cmd/main.go", false},
		{"a/?.go", "a/b.go", true},
		{"a/?.go", "a/bb.go", false},
		{"**/c.go", "c.go", true},
		{"**/c.go", "a/b/c.go", true},
		{"a/**", "a", true},
		{"a/**", "a/b/c.go", true},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"a/**/b", "a/x/y/c", false},
		{"docs/*.md", "docs/README.md", true},
		{"docs/*.md", "docs/sub/README.md", false},
	}
	for _, c := range cases {
		if got := matchPattern(c.pattern, c.path); got != c.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestValidatePattern(t *testing.T) {
	for _, bad := range []string{"a/**x/b", "a//b", "/a/b", "a/./b", "a/../b", ""} {
		if err := validatePattern(bad); err == nil {
			t.Errorf("validatePattern(%q) expected error", bad)
		}
	}
	for _, good := range []string{"a/**/b", "**", "a/*.go", "a/?.go"} {
		if err := validatePattern(good); err != nil {
			t.Errorf("validatePattern(%q) unexpected error: %v", good, err)
		}
	}
}

func TestValidatePath(t *testing.T) {
	for _, bad := range []string{"", "/a", "a//b", "a/./b", "a/../b", `a\b`} {
		if err := validatePath(bad); err == nil {
			t.Errorf("validatePath(%q) expected error", bad)
		}
	}
	if err := validatePath("a/b/c.go"); err != nil {
		t.Errorf("validatePath valid path: %v", err)
	}
}
