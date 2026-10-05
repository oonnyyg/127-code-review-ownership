package main

import "sort"

// PathReport describes ownership and coverage for one path.
type PathReport struct {
	Path        string   `json:"path"`
	Change      string   `json:"change"`
	Matched     bool     `json:"matched"`
	Owners      []string `json:"owners,omitempty"`
	ValidOwners []string `json:"valid_owners,omitempty"`
	CoveredBy   []string `json:"covered_by,omitempty"`
	Covered     bool     `json:"covered"`
	Gap         bool     `json:"gap"`
	GapReason   string   `json:"gap_reason,omitempty"`
}

// SuggestStep records one greedy pick and why it was chosen.
type SuggestStep struct {
	Reviewer      string   `json:"reviewer"`
	CoversPaths   []string `json:"covers_paths"`
	UncoveredLeft int      `json:"uncovered_left"`
}

// Report is the full JSON output.
type Report struct {
	Paths         []PathReport  `json:"paths"`
	Uncovered     []string      `json:"uncovered"`
	Gaps          []string      `json:"gaps"`
	Suggestion    []SuggestStep `json:"suggestion"`
	RemainingGaps []string      `json:"remaining_gaps"`
	Note          string        `json:"note"`
}

// buildReport resolves owners for every changed path and checks coverage
// against the given approvals.
func buildReport(r *Resolver, cs *ChangeSet, approvals []string) (*Report, error) {
	approved := map[string]bool{}
	for _, a := range approvals {
		approved[a] = true
	}
	rep := &Report{Note: "greedy suggestion is not guaranteed to be a minimum-size reviewer set"}
	for _, ch := range cs.Changes {
		if err := ch.validate(); err != nil {
			return nil, err
		}
		for _, p := range ch.Paths() {
			pr := PathReport{Path: p, Change: ch.Type}
			owners, matched, err := r.OwnersFor(p)
			if err != nil {
				return nil, err
			}
			pr.Matched = matched
			if !matched {
				pr.Gap = true
				pr.GapReason = "no ownership rule matched"
			} else {
				pr.Owners = owners
				pr.ValidOwners = r.validReviewers(owners)
				if len(pr.ValidOwners) == 0 {
					pr.Gap = true
					pr.GapReason = "no valid owner (all owners are the submitter or unavailable)"
				}
				for _, o := range pr.ValidOwners {
					if approved[o] {
						pr.CoveredBy = append(pr.CoveredBy, o)
					}
				}
				pr.Covered = len(pr.CoveredBy) > 0
			}
			if pr.Gap {
				rep.Gaps = append(rep.Gaps, p)
			} else if !pr.Covered {
				rep.Uncovered = append(rep.Uncovered, p)
			}
			rep.Paths = append(rep.Paths, pr)
		}
	}
	rep.Suggestion, rep.RemainingGaps = suggest(rep.Paths)
	return rep, nil
}

// suggest greedily picks reviewers: each step chooses the valid owner who
// covers the most still-uncovered paths; ties break by lexicographic id.
// It stops when no pick adds coverage. This is a fixed greedy heuristic,
// not a minimum-size optimum.
func suggest(paths []PathReport) ([]SuggestStep, []string) {
	uncovered := map[string]bool{}
	pathOwners := map[string][]string{}
	for _, pr := range paths {
		if pr.Gap || pr.Covered {
			continue
		}
		uncovered[pr.Path] = true
		pathOwners[pr.Path] = pr.ValidOwners
	}
	steps := []SuggestStep{}
	chosen := map[string]bool{}
	for len(uncovered) > 0 {
		count := map[string]int{}
		for p := range uncovered {
			for _, o := range pathOwners[p] {
				if !chosen[o] {
					count[o]++
				}
			}
		}
		best := ""
		bestN := 0
		for o, n := range count {
			if n > bestN || (n == bestN && n > 0 && o < best) {
				best, bestN = o, n
			}
		}
		if bestN == 0 {
			break
		}
		chosen[best] = true
		covers := []string{}
		for p := range uncovered {
			for _, o := range pathOwners[p] {
				if o == best {
					covers = append(covers, p)
					delete(uncovered, p)
					break
				}
			}
		}
		sort.Strings(covers)
		steps = append(steps, SuggestStep{Reviewer: best, CoversPaths: covers, UncoveredLeft: len(uncovered)})
	}
	remaining := make([]string, 0, len(uncovered))
	for p := range uncovered {
		remaining = append(remaining, p)
	}
	sort.Strings(remaining)
	return steps, remaining
}
