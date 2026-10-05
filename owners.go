package main

import (
	"fmt"
	"sort"
	"strings"
)

const teamPrefix = "team:"

// Resolver resolves rules and teams into per-path owner sets.
type Resolver struct {
	cfg   *Config
	teams map[string][]string
}

// NewResolver validates the config and builds a resolver.
func NewResolver(cfg *Config) (*Resolver, error) {
	r := &Resolver{cfg: cfg, teams: map[string][]string{}}
	for _, t := range cfg.Teams {
		if t.Name == "" {
			return nil, fmt.Errorf("team with empty name")
		}
		if _, dup := r.teams[t.Name]; dup {
			return nil, fmt.Errorf("duplicate team %q", t.Name)
		}
		r.teams[t.Name] = t.Members
	}
	for _, rule := range cfg.Rules {
		if err := validatePattern(rule.Pattern); err != nil {
			return nil, err
		}
		for _, o := range rule.Owners {
			if err := r.validateRef(o); err != nil {
				return nil, err
			}
		}
	}
	for name, members := range r.teams {
		for _, m := range members {
			if err := r.validateRef(m); err != nil {
				return nil, fmt.Errorf("team %q: %w", name, err)
			}
		}
	}
	return r, nil
}

func (r *Resolver) validateRef(ref string) error {
	if strings.HasPrefix(ref, teamPrefix) {
		name := strings.TrimPrefix(ref, teamPrefix)
		if _, ok := r.teams[name]; !ok {
			return fmt.Errorf("unknown team %q", name)
		}
		return nil
	}
	if !strings.HasPrefix(ref, "@") {
		return fmt.Errorf("owner %q must be \"@user\" or \"team:name\"", ref)
	}
	return nil
}

// expandTeam recursively expands a team into a deduplicated,
// sorted user list, rejecting cyclic references.
func (r *Resolver) expandTeam(name string) ([]string, error) {
	out := map[string]bool{}
	err := r.expandInto(name, map[string]bool{}, out)
	if err != nil {
		return nil, err
	}
	users := make([]string, 0, len(out))
	for u := range out {
		users = append(users, u)
	}
	sort.Strings(users)
	return users, nil
}

func (r *Resolver) expandInto(name string, visiting map[string]bool, out map[string]bool) error {
	if visiting[name] {
		return fmt.Errorf("cyclic team reference involving %q", name)
	}
	members, ok := r.teams[name]
	if !ok {
		return fmt.Errorf("unknown team %q", name)
	}
	visiting[name] = true
	defer delete(visiting, name)
	for _, m := range members {
		if strings.HasPrefix(m, teamPrefix) {
			if err := r.expandInto(strings.TrimPrefix(m, teamPrefix), visiting, out); err != nil {
				return err
			}
		} else {
			out[m] = true
		}
	}
	return nil
}

// expandRefs expands a list of user/team refs into deduplicated sorted users.
func (r *Resolver) expandRefs(refs []string) ([]string, error) {
	set := map[string]bool{}
	for _, ref := range refs {
		if strings.HasPrefix(ref, teamPrefix) {
			users, err := r.expandTeam(strings.TrimPrefix(ref, teamPrefix))
			if err != nil {
				return nil, err
			}
			for _, u := range users {
				set[u] = true
			}
		} else {
			set[ref] = true
		}
	}
	users := make([]string, 0, len(set))
	for u := range set {
		users = append(users, u)
	}
	sort.Strings(users)
	return users, nil
}

// OwnersFor returns the owners of a path. The last matching rule wins;
// matched == false means no rule matched at all.
func (r *Resolver) OwnersFor(path string) (owners []string, matched bool, err error) {
	var last *Rule
	for i := range r.cfg.Rules {
		if matchPattern(r.cfg.Rules[i].Pattern, path) {
			last = &r.cfg.Rules[i]
		}
	}
	if last == nil {
		return nil, false, nil
	}
	owners, err = r.expandRefs(last.Owners)
	if err != nil {
		return nil, true, err
	}
	return owners, true, nil
}

// validReviewers filters out the submitter and unavailable people.
func (r *Resolver) validReviewers(users []string) []string {
	blocked := map[string]bool{r.cfg.Submitter: true}
	for _, u := range r.cfg.Unavailable {
		blocked[u] = true
	}
	out := []string{}
	for _, u := range users {
		if !blocked[u] {
			out = append(out, u)
		}
	}
	return out
}
