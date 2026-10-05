package main

import (
	"reflect"
	"testing"
)

func testConfig() *Config {
	return &Config{
		Rules: []Rule{
			{Pattern: "**", Owners: []string{"team:all"}},
			{Pattern: "src/**", Owners: []string{"@alice"}},
			{Pattern: "src/legacy/**", Owners: []string{"@carol"}},
		},
		Teams: []Team{
			{Name: "all", Members: []string{"@bob", "team:backend"}},
			{Name: "backend", Members: []string{"@alice", "@bob"}},
		},
	}
}

func TestLastRuleWins(t *testing.T) {
	r, err := NewResolver(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	owners, matched, err := r.OwnersFor("src/main.go")
	if err != nil || !matched {
		t.Fatalf("owners: %v %v", matched, err)
	}
	if !reflect.DeepEqual(owners, []string{"@alice"}) {
		t.Errorf("got %v, want [@alice]", owners)
	}
	owners, _, _ = r.OwnersFor("src/legacy/x.go")
	if !reflect.DeepEqual(owners, []string{"@carol"}) {
		t.Errorf("got %v, want [@carol]", owners)
	}
}

func TestTeamOverlapDedupe(t *testing.T) {
	r, err := NewResolver(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	owners, matched, err := r.OwnersFor("README.md")
	if err != nil || !matched {
		t.Fatalf("owners: %v %v", matched, err)
	}
	// @bob appears directly and via team:backend; must be deduplicated.
	if !reflect.DeepEqual(owners, []string{"@alice", "@bob"}) {
		t.Errorf("got %v, want [@alice @bob]", owners)
	}
}

func TestNoMatch(t *testing.T) {
	cfg := &Config{Rules: []Rule{{Pattern: "src/**", Owners: []string{"@alice"}}}}
	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, matched, err := r.OwnersFor("docs/a.md")
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Error("expected no match")
	}
}

func TestUnknownTeamRejected(t *testing.T) {
	cfg := &Config{Rules: []Rule{{Pattern: "**", Owners: []string{"team:ghost"}}}}
	if _, err := NewResolver(cfg); err == nil {
		t.Error("expected error for unknown team")
	}
}

func TestCyclicTeamRejected(t *testing.T) {
	cfg := &Config{
		Rules: []Rule{{Pattern: "**", Owners: []string{"team:a"}}},
		Teams: []Team{
			{Name: "a", Members: []string{"team:b"}},
			{Name: "b", Members: []string{"team:a"}},
		},
	}
	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.OwnersFor("x.go"); err == nil {
		t.Error("expected cyclic team error")
	}
}

func TestValidReviewersExcludesSubmitterAndUnavailable(t *testing.T) {
	cfg := testConfig()
	cfg.Submitter = "@alice"
	cfg.Unavailable = []string{"@bob"}
	r, err := NewResolver(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := r.validReviewers([]string{"@alice", "@bob", "@carol"})
	if !reflect.DeepEqual(got, []string{"@carol"}) {
		t.Errorf("got %v, want [@carol]", got)
	}
}
