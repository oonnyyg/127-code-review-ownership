package main

import (
	"fmt"
	"strings"
)

// Rule maps a path pattern to a list of owners.
// Owners are "@user" identifiers or "team:name" references.
type Rule struct {
	Pattern string   `json:"pattern"`
	Owners  []string `json:"owners"`
}

// Team is a named group of members; members may reference other teams.
type Team struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// Config is the ownership configuration.
type Config struct {
	Rules       []Rule   `json:"rules"`
	Teams       []Team   `json:"teams"`
	Submitter   string   `json:"submitter"`
	Unavailable []string `json:"unavailable"`
}

// Change describes a single change. Type is one of
// "add", "modify", "delete", "rename".
type Change struct {
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
}

// ChangeSet is the JSON change manifest.
type ChangeSet struct {
	Changes []Change `json:"changes"`
}

// Approvals lists the approver identifiers to check.
type Approvals struct {
	Approvals []string `json:"approvals"`
}

// validatePath enforces repo-root-relative paths using forward slashes,
// with no empty segments and no standalone "." or ".." segments.
func validatePath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path %q must use forward slashes", p)
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("path %q must be relative to the repository root", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return fmt.Errorf("path %q contains an empty segment", p)
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("path %q contains a %q segment", p, seg)
		}
	}
	return nil
}

// validate checks the change entry and its paths.
func (c Change) validate() error {
	switch c.Type {
	case "add", "modify", "delete":
		if c.Path == "" {
			return fmt.Errorf("change of type %q requires path", c.Type)
		}
		return validatePath(c.Path)
	case "rename":
		if c.OldPath == "" || c.NewPath == "" {
			return fmt.Errorf("rename requires old_path and new_path")
		}
		if err := validatePath(c.OldPath); err != nil {
			return err
		}
		return validatePath(c.NewPath)
	default:
		return fmt.Errorf("unknown change type %q", c.Type)
	}
}

// Paths returns every path that must be checked for this change.
func (c Change) Paths() []string {
	if c.Type == "rename" {
		return []string{c.OldPath, c.NewPath}
	}
	return []string{c.Path}
}
