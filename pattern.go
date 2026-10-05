package main

import (
	"fmt"
	"strings"
)

// matchPattern reports whether pattern matches the whole path.
//
// Pattern language (NOT full CODEOWNERS semantics):
//   - "*" matches any run of characters within a single segment
//   - "?" matches exactly one character within a segment
//   - "**" as a whole segment matches zero or more segments
//   - every other character is literal
func matchPattern(pattern, path string) bool {
	psegs := strings.Split(pattern, "/")
	segs := strings.Split(path, "/")
	return matchSegments(psegs, segs)
}

func matchSegments(pat, segs []string) bool {
	if len(pat) == 0 {
		return len(segs) == 0
	}
	if pat[0] == "**" {
		// "**" consumes zero or more segments.
		for i := 0; i <= len(segs); i++ {
			if matchSegments(pat[1:], segs[i:]) {
				return true
			}
		}
		return false
	}
	if len(segs) == 0 {
		return false
	}
	if !matchSegment(pat[0], segs[0]) {
		return false
	}
	return matchSegments(pat[1:], segs[1:])
}

// matchSegment matches one pattern segment against one path segment
// using "*" (any run) and "?" (single char) wildcards.
func matchSegment(pat, s string) bool {
	if pat == "" {
		return s == ""
	}
	switch pat[0] {
	case '*':
		for i := 0; i <= len(s); i++ {
			if matchSegment(pat[1:], s[i:]) {
				return true
			}
		}
		return false
	case '?':
		if s == "" {
			return false
		}
		return matchSegment(pat[1:], s[1:])
	default:
		if s == "" || s[0] != pat[0] {
			return false
		}
		return matchSegment(pat[1:], s[1:])
	}
}

// validatePattern ensures the pattern is a valid relative path pattern
// and that "**" only appears as a whole segment.
func validatePattern(p string) error {
	if err := validatePath(p); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	for _, seg := range strings.Split(p, "/") {
		if strings.Contains(seg, "**") && seg != "**" {
			return fmt.Errorf("pattern %q: \"**\" must occupy a whole segment", p)
		}
	}
	return nil
}
