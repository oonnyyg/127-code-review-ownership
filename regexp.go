package review

import (
	"regexp"
	"sort"
)

// regexpQuoteRune escapes a rune for literal use inside a regexp.
func regexpQuoteRune(r rune) string {
	return regexp.QuoteMeta(string(r))
}

// matchRegexp matches an unanchored single-segment regexp against text.
func matchRegexp(re, text string) bool {
	matched, err := regexp.MatchString("^(?:"+re+")$", text)
	if err != nil {
		// re is built from QuoteMeta and fixed fragments; unreachable in
		// practice, but fail closed rather than silently matching.
		return false
	}
	return matched
}

// sortedKeys returns the keys of m in ascending order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
