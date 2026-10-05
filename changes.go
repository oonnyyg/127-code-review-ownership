package review

import (
	"encoding/json"
	"fmt"
	"io"
)

// ChangeKind is the kind of a single change entry.
type ChangeKind string

const (
	Add    ChangeKind = "add"
	Modify ChangeKind = "modify"
	Delete ChangeKind = "delete"
	Rename ChangeKind = "rename"
)

// ChangeJSON is one entry of the JSON change list.
type ChangeJSON struct {
	Kind ChangeKind `json:"kind"`
	Path string     `json:"path,omitempty"`
	Old  string     `json:"old_path,omitempty"`
	New  string     `json:"new_path,omitempty"`
}

// ChangeSetJSON is the decoded change list document.
type ChangeSetJSON struct {
	Author    string       `json:"author"`
	Approvals []string     `json:"approvals"`
	Changes   []ChangeJSON `json:"changes"`
}

// PathEntry is one repository-relative path that must be reviewed,
// together with the change kind and the change index it came from.
// A rename contributes two entries: the old path and the new path.
type PathEntry struct {
	ChangeIndex int
	Kind        ChangeKind
	Path        string
}

// ChangeSet is a validated change list.
type ChangeSet struct {
	Author    string
	Approvals []string
	Paths     []PathEntry
}

// LoadChangeSet decodes and validates a JSON change list from r.
func LoadChangeSet(r io.Reader) (*ChangeSet, error) {
	var doc ChangeSetJSON
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid change list JSON: %w", err)
	}
	return NewChangeSet(doc)
}

// NewChangeSet validates a decoded change list document.
func NewChangeSet(doc ChangeSetJSON) (*ChangeSet, error) {
	cs := &ChangeSet{Author: doc.Author, Approvals: doc.Approvals}
	if cs.Author == "" {
		return nil, fmt.Errorf("change list: author is required")
	}
	for i, ch := range doc.Changes {
		switch ch.Kind {
		case Add, Modify, Delete:
			if ch.Path == "" {
				return nil, fmt.Errorf("change %d (%s): path is required", i, ch.Kind)
			}
			if err := ValidatePath(ch.Path); err != nil {
				return nil, fmt.Errorf("change %d (%s): %w", i, ch.Kind, err)
			}
			cs.Paths = append(cs.Paths, PathEntry{ChangeIndex: i, Kind: ch.Kind, Path: ch.Path})
		case Rename:
			if ch.Old == "" || ch.New == "" {
				return nil, fmt.Errorf("change %d (rename): old_path and new_path are required", i)
			}
			if err := ValidatePath(ch.Old); err != nil {
				return nil, fmt.Errorf("change %d (rename): old_path: %w", i, err)
			}
			if err := ValidatePath(ch.New); err != nil {
				return nil, fmt.Errorf("change %d (rename): new_path: %w", i, err)
			}
			cs.Paths = append(cs.Paths,
				PathEntry{ChangeIndex: i, Kind: Rename, Path: ch.Old},
				PathEntry{ChangeIndex: i, Kind: Rename, Path: ch.New},
			)
		default:
			return nil, fmt.Errorf("change %d: unknown kind %q", i, ch.Kind)
		}
	}
	if len(cs.Paths) == 0 {
		return nil, fmt.Errorf("change list: at least one change is required")
	}
	return cs, nil
}
