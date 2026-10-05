package review

import (
	"reflect"
	"testing"
)

func TestApprovalsOnlyCountFromOwners(t *testing.T) {
	cfg := testConfig(t, baseConfig)
	cs := testChanges(t, `{
	  "author": "dave",
	  "approvals": ["frank", "carol"],
	  "changes": [
	    {"kind": "modify", "path": "src/api.go"},
	    {"kind": "add", "path": "notes.txt"}
	  ]
	}`)
	rep := Resolve(cfg, cs)
	// carol approved but is not an owner of src/**: no coverage.
	if rep.Paths[0].Covered {
		t.Errorf("src/api.go covered by non-owner approval; approvals=%v", cs.Approvals)
	}
	// frank owns ** and his approval covers notes.txt.
	if !rep.Paths[1].Covered || !reflect.DeepEqual(rep.Paths[1].ApprovedBy, []string{"frank"}) {
		t.Errorf("notes.txt coverage = %+v, want covered by frank", rep.Paths[1])
	}
}

func TestAuthorCannotBeReviewer(t *testing.T) {
	cfg := testConfig(t, baseConfig)
	cs := testChanges(t, `{
	  "author": "carol",
	  "approvals": ["carol"],
	  "changes": [{"kind": "modify", "path": "src/legacy/old.go"}]
	}`)
	rep := Resolve(cfg, cs)
	p := rep.Paths[0]
	if len(p.Eligible) != 0 || !p.Gap || p.Covered {
		t.Errorf("author-only ownership should be a gap, got %+v", p)
	}
	if !reflect.DeepEqual(rep.Gaps, []string{"src/legacy/old.go"}) {
		t.Errorf("gaps = %v", rep.Gaps)
	}
}

func TestUnmatchedPathMarked(t *testing.T) {
	cfg := testConfig(t, `{
	  "people": {"alice": {}},
	  "teams": {},
	  "rules": [{"pattern": "src/**", "owners": ["alice"]}]
	}`)
	cs := testChanges(t, `{
	  "author": "bob",
	  "changes": [{"kind": "delete", "path": "docs/readme.md"}]
	}`)
	rep := Resolve(cfg, cs)
	p := rep.Paths[0]
	if p.Matched || p.Covered {
		t.Errorf("unmatched path should be marked, got %+v", p)
	}
	if !reflect.DeepEqual(rep.RemainingAfterSuggestion, []string{"docs/readme.md"}) {
		t.Errorf("remaining gaps = %v", rep.RemainingAfterSuggestion)
	}
}

func TestGreedySuggestionDeterministic(t *testing.T) {
	cfg := testConfig(t, `{
	  "people": {"a": {}, "b": {}, "c": {}, "d": {}},
	  "teams": {},
	  "rules": [
	    {"pattern": "x/**", "owners": ["a", "b"]},
	    {"pattern": "y/**", "owners": ["b", "c"]},
	    {"pattern": "z/**", "owners": ["c"]}
	  ]
	}`)
	cs := testChanges(t, `{
	  "author": "d",
	  "changes": [
	    {"kind": "modify", "path": "x/1"},
	    {"kind": "modify", "path": "x/2"},
	    {"kind": "modify", "path": "y/1"},
	    {"kind": "modify", "path": "z/1"}
	  ]
	}`)
	rep := Resolve(cfg, cs)
	// b covers 3 paths first; then c covers z/1. Greedy, not minimal.
	if !reflect.DeepEqual(rep.SuggestedNames, []string{"b", "c"}) {
		t.Fatalf("suggested = %v, want [b c]", rep.SuggestedNames)
	}
	if !reflect.DeepEqual(rep.Suggestion[0].Covers, []string{"x/1", "x/2", "y/1"}) {
		t.Errorf("first pick covers = %v", rep.Suggestion[0].Covers)
	}
	if !reflect.DeepEqual(rep.Suggestion[1].Covers, []string{"z/1"}) {
		t.Errorf("second pick covers = %v", rep.Suggestion[1].Covers)
	}
	if len(rep.RemainingAfterSuggestion) != 0 {
		t.Errorf("remaining = %v, want none", rep.RemainingAfterSuggestion)
	}
}

func TestGreedyTieBreaksByPersonID(t *testing.T) {
	cfg := testConfig(t, `{
	  "people": {"ann": {}, "amy": {}},
	  "teams": {},
	  "rules": [{"pattern": "**", "owners": ["amy", "ann"]}]
	}`)
	cs := testChanges(t, `{
	  "author": "zed",
	  "changes": [{"kind": "modify", "path": "f.go"}]
	}`)
	rep := Resolve(cfg, cs)
	if !reflect.DeepEqual(rep.SuggestedNames, []string{"amy"}) {
		t.Errorf("tie should pick lexicographically smaller ID, got %v", rep.SuggestedNames)
	}
}
