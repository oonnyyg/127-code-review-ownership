package review

import "testing"

func TestValidatePath(t *testing.T) {
	valid := []string{"a", "a/b/c.go", "x-y/z_1.txt"}
	for _, p := range valid {
		if err := ValidatePath(p); err != nil {
			t.Errorf("ValidatePath(%q) = %v, want nil", p, err)
		}
	}
	invalid := []string{"", "/a", "a//b", "a/./b", "a/../b", "..", ".", `a\b`}
	for _, p := range invalid {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) = nil, want error", p)
		}
	}
}

func TestPatternMatch(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"a/b.go", "a/b.go", true},
		{"a/b.go", "a/c.go", false},
		{"a/*.go", "a/b.go", true},
		{"a/*.go", "a/x/b.go", false}, // * stays within one segment
		{"a/*.go", "a/b.txt", false},
		{"a/b?.go", "a/b1.go", true}, // ? matches one character
		{"a/b?.go", "a/b12.go", false},
		{"docs/**", "docs/a/b/c.md", true},
		{"docs/**", "docs", true}, // ** matches zero segments
		{"**/main.go", "main.go", true},
		{"**/main.go", "cmd/app/main.go", true},
		{"src/**/test/*", "src/test/x", true},
		{"src/**/test/*", "src/a/b/test/x", true},
		{"src/**/test/*", "src/a/b/prod/x", false},
		{"a+b.go", "a+b.go", true}, // literal regexp metacharacters
		{"a+b.go", "ab.go", false},
		{"a/*.go", "a/b.go/..", false},
	}
	for _, c := range cases {
		p, err := CompilePattern(c.pattern)
		if err != nil {
			t.Fatalf("CompilePattern(%q): %v", c.pattern, err)
		}
		if got := p.Match(c.path); got != c.want {
			t.Errorf("pattern %q vs path %q = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestCompilePatternRejectsBadDoubleStar(t *testing.T) {
	for _, bad := range []string{"a/**/b/**", "a/**x", "a/x**/b", "**a/b"} {
		if bad == "a/**/b/**" { // legal: ** as whole segments
			if _, err := CompilePattern(bad); err != nil {
				t.Errorf("CompilePattern(%q) = %v, want nil", bad, err)
			}
			continue
		}
		if _, err := CompilePattern(bad); err == nil {
			t.Errorf("CompilePattern(%q) = nil, want error", bad)
		}
	}
}
