// Package review implements an offline code-review ownership tool.
// It matches changed paths against ordered ownership rules, expands
// teams from a roster, checks approvals, and suggests reviewers.
package review

import (
	"fmt"
	"strings"
)

// ValidatePath reports whether p is a valid repository-relative path:
// forward slashes only, no empty segments, and no "." or ".." segments.
func ValidatePath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path %q: backslashes are not allowed, use forward slashes", p)
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("path %q: must be relative to the repository root", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return fmt.Errorf("path %q: contains an empty segment", p)
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("path %q: segment %q is not allowed", p, seg)
		}
	}
	return nil
}

// segment is one compiled path segment of a pattern.
type segment struct {
	doubleStar bool   // true when the segment is exactly "**"
	literal    string // set when the segment has no wildcards
	re         string // compiled single-segment regexp (without anchors)
}

// Pattern is a compiled ownership path pattern.
type Pattern struct {
	raw  string
	segs []segment
}

// CompilePattern validates and compiles a pattern. The pattern language:
//   - "*" matches any characters within one segment,
//   - "?" matches exactly one character within one segment,
//   - "**" as a whole segment matches zero or more whole segments,
//   - every other character is literal.
func CompilePattern(raw string) (*Pattern, error) {
	if err := ValidatePath(raw); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	p := &Pattern{raw: raw}
	for _, seg := range strings.Split(raw, "/") {
		if seg == "**" {
			p.segs = append(p.segs, segment{doubleStar: true})
			continue
		}
		if strings.Contains(seg, "**") {
			return nil, fmt.Errorf("pattern %q: %q is only allowed as a whole segment", raw, "**")
		}
		if !strings.ContainsAny(seg, "*?") {
			p.segs = append(p.segs, segment{literal: seg})
			continue
		}
		var b strings.Builder
		for _, r := range seg {
			switch r {
			case '*':
				b.WriteString(`[^/]*`)
			case '?':
				b.WriteString(`[^/]`)
			default:
				b.WriteString(regexpQuoteRune(r))
			}
		}
		p.segs = append(p.segs, segment{re: b.String()})
	}
	return p, nil
}

// Raw returns the original pattern text.
func (p *Pattern) Raw() string { return p.raw }

// Match reports whether the pattern matches the whole path.
// The path must already be validated.
func (p *Pattern) Match(path string) bool {
	return matchSegs(p.segs, strings.Split(path, "/"))
}

func matchSegs(pat []segment, path []string) bool {
	if len(pat) == 0 {
		return len(path) == 0
	}
	head := pat[0]
	if head.doubleStar {
		// "**" matches zero or more whole segments.
		for skip := 0; skip <= len(path); skip++ {
			if matchSegs(pat[1:], path[skip:]) {
				return true
			}
		}
		return false
	}
	if len(path) == 0 {
		return false
	}
	if !matchOne(head, path[0]) {
		return false
	}
	return matchSegs(pat[1:], path[1:])
}

func matchOne(s segment, text string) bool {
	if s.literal != "" {
		return s.literal == text
	}
	return matchRegexp(s.re, text)
}
