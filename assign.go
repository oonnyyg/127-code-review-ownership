package review

import "sort"

// PathReport describes ownership resolution for one changed path.
type PathReport struct {
	Path        string     `json:"path"`
	Kind        ChangeKind `json:"kind"`
	ChangeIndex int        `json:"change_index"`
	Matched     bool       `json:"matched"`
	RuleIndex   int        `json:"rule_index"` // -1 when unmatched
	Pattern     string     `json:"pattern,omitempty"`
	// Owners is every responsible person after team expansion.
	Owners []string `json:"owners,omitempty"`
	// Eligible excludes the author and unavailable people; only these
	// may act as reviewers for this path.
	Eligible []string `json:"eligible_reviewers,omitempty"`
	// ApprovedBy lists given approvals that count for this path.
	ApprovedBy []string `json:"approved_by,omitempty"`
	Covered    bool     `json:"covered"`
	// Gap is true when the path has no eligible reviewer at all.
	Gap bool `json:"gap"`
}

// SuggestionStep records one greedy pick.
type SuggestionStep struct {
	Person string   `json:"person"`
	Covers []string `json:"covers"`
}

// Report is the full analysis result.
type Report struct {
	Author string       `json:"author"`
	Paths  []PathReport `json:"paths"`
	// Gaps lists paths with no eligible reviewer.
	Gaps []string `json:"gaps"`
	// Uncovered lists paths not approved by any eligible owner.
	Uncovered []string `json:"uncovered"`
	// Suggestion is the greedy reviewer selection; it is deterministic
	// but not guaranteed to be a minimum-size set.
	Suggestion     []SuggestionStep `json:"suggestion"`
	SuggestedNames []string         `json:"suggested_reviewers"`
	// RemainingAfterSuggestion lists paths no eligible reviewer can cover.
	RemainingAfterSuggestion []string `json:"remaining_gaps"`
}

// Resolve matches every changed path against the ordered rules and
// evaluates the given approvals. The last matching rule wins; paths
// with no matching rule are explicitly marked unmatched.
func Resolve(cfg *Config, cs *ChangeSet) *Report {
	rep := &Report{Author: cs.Author}
	approvals := map[string]bool{}
	for _, a := range cs.Approvals {
		approvals[a] = true
	}
	for _, pe := range cs.Paths {
		pr := PathReport{Path: pe.Path, Kind: pe.Kind, ChangeIndex: pe.ChangeIndex, RuleIndex: -1}
		ruleIdx := -1
		for i, rule := range cfg.Rules {
			if rule.Pattern.Match(pe.Path) {
				ruleIdx = i
			}
		}
		if ruleIdx >= 0 {
			rule := cfg.Rules[ruleIdx]
			pr.Matched = true
			pr.RuleIndex = ruleIdx
			pr.Pattern = rule.Pattern.Raw()
			owners := map[string]bool{}
			for _, ref := range rule.Owners {
				if ref.IsTeam() {
					members, err := cfg.ExpandTeam(ref.Name())
					if err != nil {
						// Teams were validated at load time.
						continue
					}
					for _, m := range members {
						owners[m] = true
					}
				} else {
					owners[ref.Name()] = true
				}
			}
			pr.Owners = sortedKeys(owners)
			eligible := map[string]bool{}
			for _, o := range pr.Owners {
				if o == cs.Author {
					continue
				}
				if !cfg.Available(o) {
					continue
				}
				eligible[o] = true
			}
			pr.Eligible = sortedKeys(eligible)
			for _, o := range pr.Eligible {
				if approvals[o] {
					pr.ApprovedBy = append(pr.ApprovedBy, o)
				}
			}
		}
		pr.Covered = len(pr.ApprovedBy) > 0
		pr.Gap = pr.Matched && len(pr.Eligible) == 0
		if pr.Gap {
			rep.Gaps = append(rep.Gaps, pr.Path)
		}
		if !pr.Covered {
			rep.Uncovered = append(rep.Uncovered, pr.Path)
		}
		rep.Paths = append(rep.Paths, pr)
	}
	rep.Suggestion, rep.RemainingAfterSuggestion = Suggest(rep)
	for _, s := range rep.Suggestion {
		rep.SuggestedNames = append(rep.SuggestedNames, s.Person)
	}
	return rep
}

// Suggest picks reviewers with a fixed greedy rule: repeatedly choose
// the person who covers the most still-uncovered paths, breaking ties
// by ascending person ID, until nobody adds coverage. The result is
// deterministic but not guaranteed to be a minimum-size set.
func Suggest(rep *Report) ([]SuggestionStep, []string) {
	uncovered := map[string]bool{}
	coverage := map[string][]string{} // person -> coverable paths
	for _, pr := range rep.Paths {
		if len(pr.Eligible) == 0 {
			continue // unmatched or gap: nobody can cover it
		}
		if pr.Covered {
			continue // already approved by an eligible owner
		}
		uncovered[pr.Path] = true
		for _, p := range pr.Eligible {
			coverage[p] = append(coverage[p], pr.Path)
		}
	}
	var steps []SuggestionStep
	for len(uncovered) > 0 {
		best := ""
		var bestCovers []string
		for _, person := range sortedKeys(coverage) {
			var covers []string
			for _, path := range coverage[person] {
				if uncovered[path] {
					covers = append(covers, path)
				}
			}
			if len(covers) > len(bestCovers) {
				best, bestCovers = person, covers
			}
		}
		if len(bestCovers) == 0 {
			break
		}
		sort.Strings(bestCovers)
		steps = append(steps, SuggestionStep{Person: best, Covers: bestCovers})
		for _, path := range bestCovers {
			delete(uncovered, path)
		}
	}
	var remaining []string
	for _, pr := range rep.Paths {
		if len(pr.Eligible) == 0 {
			remaining = append(remaining, pr.Path)
		}
	}
	return steps, remaining
}
