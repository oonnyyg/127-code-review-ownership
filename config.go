package review

import (
	"encoding/json"
	"fmt"
	"io"
)

// OwnerRef is a single owner entry in a rule: either a person ID
// (e.g. "alice") or a team reference prefixed with "@" (e.g. "@backend").
type OwnerRef string

// IsTeam reports whether the reference names a team.
func (o OwnerRef) IsTeam() bool {
	return len(o) > 1 && o[0] == '@'
}

// Name returns the referenced person or team name without any "@" prefix.
func (o OwnerRef) Name() string {
	if o.IsTeam() {
		return string(o[1:])
	}
	return string(o)
}

// RuleJSON is one ordered ownership rule.
type RuleJSON struct {
	Pattern string   `json:"pattern"`
	Owners  []string `json:"owners"`
}

// TeamJSON is a roster team; members are person IDs, teams are nested
// team names (without "@").
type TeamJSON struct {
	Members []string `json:"members"`
	Teams   []string `json:"teams"`
}

// PersonJSON is a roster entry for one person.
type PersonJSON struct {
	Unavailable bool `json:"unavailable"`
}

// ConfigJSON is the decoded JSON configuration document.
type ConfigJSON struct {
	Rules  []RuleJSON            `json:"rules"`
	Teams  map[string]TeamJSON   `json:"teams"`
	People map[string]PersonJSON `json:"people"`
}

// Rule is a compiled ownership rule.
type Rule struct {
	Pattern *Pattern
	Owners  []OwnerRef
}

// Config is the validated, compiled configuration.
type Config struct {
	Rules  []Rule
	teams  map[string]TeamJSON
	people map[string]PersonJSON
}

// LoadConfig decodes and validates a JSON configuration from r.
func LoadConfig(r io.Reader) (*Config, error) {
	var doc ConfigJSON
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid config JSON: %w", err)
	}
	return NewConfig(doc)
}

// NewConfig validates a decoded configuration document.
func NewConfig(doc ConfigJSON) (*Config, error) {
	cfg := &Config{teams: doc.Teams, people: doc.People}
	if cfg.teams == nil {
		cfg.teams = map[string]TeamJSON{}
	}
	if cfg.people == nil {
		cfg.people = map[string]PersonJSON{}
	}
	for name, team := range cfg.teams {
		if name == "" {
			return nil, fmt.Errorf("team name must not be empty")
		}
		for _, m := range team.Members {
			if _, ok := cfg.people[m]; !ok {
				return nil, fmt.Errorf("team %q: unknown person %q", name, m)
			}
		}
		for _, sub := range team.Teams {
			if _, ok := cfg.teams[sub]; !ok {
				return nil, fmt.Errorf("team %q: unknown nested team %q", name, sub)
			}
		}
	}
	for i, rj := range doc.Rules {
		pat, err := CompilePattern(rj.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
		if len(rj.Owners) == 0 {
			return nil, fmt.Errorf("rule %d (%q): at least one owner is required", i, rj.Pattern)
		}
		rule := Rule{Pattern: pat}
		for _, o := range rj.Owners {
			ref := OwnerRef(o)
			if ref.IsTeam() {
				if _, ok := cfg.teams[ref.Name()]; !ok {
					return nil, fmt.Errorf("rule %d (%q): unknown team %q", i, rj.Pattern, o)
				}
			} else {
				if _, ok := cfg.people[ref.Name()]; !ok {
					return nil, fmt.Errorf("rule %d (%q): unknown person %q", i, rj.Pattern, o)
				}
			}
			rule.Owners = append(rule.Owners, ref)
		}
		cfg.Rules = append(cfg.Rules, rule)
	}
	// Validate team graph eagerly so cycles are rejected at load time.
	for name := range cfg.teams {
		if _, err := cfg.ExpandTeam(name); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// ExpandTeam returns the deduplicated, sorted person IDs of a team,
// recursively expanding nested teams. Cyclic nesting is rejected.
func (c *Config) ExpandTeam(name string) ([]string, error) {
	if _, ok := c.teams[name]; !ok {
		return nil, fmt.Errorf("unknown team %q", name)
	}
	out := map[string]bool{}
	if err := c.expandTeam(name, map[string]bool{name: true}, out); err != nil {
		return nil, err
	}
	return sortedKeys(out), nil
}

func (c *Config) expandTeam(name string, visiting, out map[string]bool) error {
	team := c.teams[name]
	for _, m := range team.Members {
		out[m] = true
	}
	for _, sub := range team.Teams {
		if visiting[sub] {
			return fmt.Errorf("team %q: cyclic team nesting via %q", name, sub)
		}
		visiting[sub] = true
		if err := c.expandTeam(sub, visiting, out); err != nil {
			return err
		}
		delete(visiting, sub)
	}
	return nil
}

// Available reports whether the person exists and is not unavailable.
func (c *Config) Available(person string) bool {
	p, ok := c.people[person]
	return ok && !p.Unavailable
}
