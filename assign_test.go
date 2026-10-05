package review

import (
	"reflect"
	"strings"
	"testing"
)

func testConfig(t *testing.T, jsonText string) *Config {
	t.Helper()
	cfg, err := LoadConfig(strings.NewReader(jsonText))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

func testChanges(t *testing.T, jsonText string) *ChangeSet {
	t.Helper()
	cs, err := LoadChangeSet(strings.NewReader(jsonText))
	if err != nil {
		t.Fatalf("LoadChangeSet: %v", err)
	}
	return cs
}

const baseConfig = `{
  "people": {
    "alice": {}, "bob": {}, "carol": {}, "dave": {},
    "erin": {"unavailable": true}, "frank": {}
  },
  "teams": {
    "backend":  {"members": ["alice", "bob"]},
    "frontend": {"members": ["bob", "carol"]},
    "all-devs": {"members": ["dave"], "teams": ["backend", "frontend"]}
  },
  "rules": [
    {"pattern": "**", "owners": ["frank"]},
    {"pattern": "src/**", "owners": ["@backend"]},
    {"pattern": "src/legacy/**", "owners": ["carol"]},
    {"pattern": "web/**", "owners": ["@frontend", "erin"]}
  ]
}`

func TestLastMatchingRuleWins(t *testing.T) {
	cfg := testConfig(t, baseConfig)
	cs := testChanges(t, `{
	  "author": "dave",
	  "changes": [
	    {"kind": "modify", "path": "src/legacy/old.go"},
	    {"kind": "modify", "path": "src/new.go"},
	    {"kind": "modify", "path": "README.md"}
	  ]
	}`)
	rep := Resolve(cfg, cs)
	got := []string{rep.Paths[0].Pattern, rep.Paths[1].Pattern, rep.Paths[2].Pattern}
	want := []string{"src/legacy/**", "src/**", "**"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("matched patterns = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(rep.Paths[0].Owners, []string{"carol"}) {
		t.Errorf("legacy owners = %v, want [carol]", rep.Paths[0].Owners)
	}
}

func TestRenameChecksBothPathsAcrossDirectories(t *testing.T) {
	cfg := testConfig(t, baseConfig)
	cs := testChanges(t, `{
	  "author": "dave",
	  "changes": [
	    {"kind": "rename", "old_path": "src/api.go", "new_path": "web/api.ts"}
	  ]
	}`)
	rep := Resolve(cfg, cs)
	if len(rep.Paths) != 2 {
		t.Fatalf("rename produced %d path entries, want 2", len(rep.Paths))
	}
	old, newP := rep.Paths[0], rep.Paths[1]
	if old.Path != "src/api.go" || !reflect.DeepEqual(old.Eligible, []string{"alice", "bob"}) {
		t.Errorf("old path entry = %+v", old)
	}
	// erin is unavailable and must be excluded from eligible reviewers.
	if newP.Path != "web/api.ts" || !reflect.DeepEqual(newP.Eligible, []string{"bob", "carol"}) {
		t.Errorf("new path entry = %+v", newP)
	}
	if !reflect.DeepEqual(newP.Owners, []string{"bob", "carol", "erin"}) {
		t.Errorf("new path owners = %v, want erin listed as owner", newP.Owners)
	}
}

func TestOverlappingTeamsExpandAndDeduplicate(t *testing.T) {
	cfg := testConfig(t, baseConfig)
	members, err := cfg.ExpandTeam("all-devs")
	if err != nil {
		t.Fatalf("ExpandTeam: %v", err)
	}
	want := []string{"alice", "bob", "carol", "dave"}
	if !reflect.DeepEqual(members, want) {
		t.Errorf("ExpandTeam(all-devs) = %v, want %v (deduplicated)", members, want)
	}
}

func TestCyclicAndUnknownTeamsRejected(t *testing.T) {
	cyclic := `{
	  "people": {"alice": {}},
	  "teams": {
	    "a": {"teams": ["b"]},
	    "b": {"teams": ["a"]}
	  },
	  "rules": [{"pattern": "**", "owners": ["@a"]}]
	}`
	if _, err := LoadConfig(strings.NewReader(cyclic)); err == nil {
		t.Error("cyclic team nesting accepted, want error")
	}
	unknown := `{
	  "people": {"alice": {}},
	  "teams": {},
	  "rules": [{"pattern": "**", "owners": ["@ghost"]}]
	}`
	if _, err := LoadConfig(strings.NewReader(unknown)); err == nil {
		t.Error("unknown team accepted, want error")
	}
}
