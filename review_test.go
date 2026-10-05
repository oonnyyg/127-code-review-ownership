package main

import "testing"

func reviewConfig() *Config {
	return &Config{
		Rules: []Rule{
			{Pattern: "src/**", Owners: []string{"@alice", "@bob"}},
			{Pattern: "docs/**", Owners: []string{"@carol"}},
			{Pattern: "web/**", Owners: []string{"team:web"}},
		},
		Teams: []Team{
			{Name: "web", Members: []string{"@alice", "@dave"}},
		},
		Submitter: "@dave",
	}
}

func build(t *testing.T, cfg *Config, cs *ChangeSet, approvals []string) *Report {
	t.Helper()
	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := buildReport(r, cs, approvals)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestRenameChecksBothPaths(t *testing.T) {
	cs := &ChangeSet{Changes: []Change{
		{Type: "rename", OldPath: "src/old.go", NewPath: "docs/new.go"},
	}}
	rep := build(t, reviewConfig(), cs, nil)
	if len(rep.Paths) != 2 {
		t.Fatalf("expected 2 path reports, got %d", len(rep.Paths))
	}
	if rep.Paths[0].Path != "src/old.go" || rep.Paths[1].Path != "docs/new.go" {
		t.Errorf("unexpected paths: %v, %v", rep.Paths[0].Path, rep.Paths[1].Path)
	}
	if got := rep.Paths[0].Owners; len(got) != 2 || got[0] != "@alice" || got[1] != "@bob" {
		t.Errorf("old path owners = %v", got)
	}
	if got := rep.Paths[1].Owners; len(got) != 1 || got[0] != "@carol" {
		t.Errorf("new path owners = %v", got)
	}
}

func TestCoverageAndNonOwnerApproval(t *testing.T) {
	cs := &ChangeSet{Changes: []Change{
		{Type: "modify", Path: "src/a.go"},
		{Type: "add", Path: "docs/b.md"},
	}}
	// @carol is not an owner of src/a.go; approval must not cover it.
	rep := build(t, reviewConfig(), cs, []string{"@carol"})
	if rep.Paths[0].Covered {
		t.Error("non-owner approval must not cover src/a.go")
	}
	if !rep.Paths[1].Covered {
		t.Error("docs/b.md should be covered by @carol")
	}
	if len(rep.Uncovered) != 1 || rep.Uncovered[0] != "src/a.go" {
		t.Errorf("uncovered = %v", rep.Uncovered)
	}
}

func TestGapWhenNoValidOwner(t *testing.T) {
	cfg := reviewConfig()
	cfg.Unavailable = []string{"@carol"}
	cs := &ChangeSet{Changes: []Change{
		{Type: "delete", Path: "docs/b.md"},
		{Type: "add", Path: "other/c.txt"},
	}}
	rep := build(t, cfg, cs, nil)
	if len(rep.Gaps) != 2 {
		t.Fatalf("gaps = %v", rep.Gaps)
	}
	if rep.Paths[0].GapReason == "" || rep.Paths[1].GapReason == "" {
		t.Error("gap reasons must be set")
	}
}

func TestGreedySuggestion(t *testing.T) {
	cs := &ChangeSet{Changes: []Change{
		{Type: "modify", Path: "src/a.go"},
		{Type: "modify", Path: "src/b.go"},
		{Type: "modify", Path: "web/c.js"},
	}}
	rep := build(t, reviewConfig(), cs, nil)
	// @alice covers all three paths (web via team, @dave excluded as submitter).
	if len(rep.Suggestion) != 1 || rep.Suggestion[0].Reviewer != "@alice" {
		t.Fatalf("suggestion = %+v", rep.Suggestion)
	}
	if rep.Suggestion[0].UncoveredLeft != 0 {
		t.Errorf("uncovered left = %d", rep.Suggestion[0].UncoveredLeft)
	}
	if len(rep.RemainingGaps) != 0 {
		t.Errorf("remaining gaps = %v", rep.RemainingGaps)
	}
}

func TestGreedyTieBreakLexicographic(t *testing.T) {
	cfg := &Config{
		Rules: []Rule{
			{Pattern: "a/**", Owners: []string{"@zoe", "@amy"}},
			{Pattern: "b/**", Owners: []string{"@zoe", "@amy"}},
		},
	}
	cs := &ChangeSet{Changes: []Change{
		{Type: "modify", Path: "a/x"},
		{Type: "modify", Path: "b/y"},
	}}
	rep := build(t, cfg, cs, nil)
	// Both cover 2 paths; the tie breaks to the lexicographically smaller id.
	if len(rep.Suggestion) != 1 {
		t.Fatalf("suggestion = %+v", rep.Suggestion)
	}
	if rep.Suggestion[0].Reviewer != "@amy" {
		t.Errorf("first pick = %v, want @amy", rep.Suggestion[0].Reviewer)
	}
}
