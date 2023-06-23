package difflint

import (
	"log"
	"os"

	"github.com/pkg/errors"
)

// Target represents a file or range of code that must be present in the diff
// if a diff hunk is present.
type Target struct {
	// File specifier expected to contain a diff hunk.
	File *string

	// ID is the ID of the range of code in which a diff hunk intersects.
	ID *string
}

// A rule says that file or range of code must be present in the diff if another range is present.
type Rule struct {
	// Hunk is the diff hunk that must be present in the diff.
	Hunk Hunk

	// Targets are the files or ranges of code that must be present in the diff if the hunk is present.
	Targets []Target

	// Present is true if the change is present in the diff from which the rules were parsed.
	Present bool

	// ID is an optional, unique identifier for the rule.
	ID *string
}

// RulesMapFromHunks parses rules from the given hunks by file name and
// returns the map of rules.
func RulesMapFromHunks(hunks []Hunk, options LintOptions) (map[string][]Rule, map[string]struct{}, error) {
	// Separate hunks by file name and construct a set of all the target keys that exist.
	targetsMap := make(map[string]struct{}, len(hunks))
	rangesMap := make(map[string][]Range, len(hunks))
	for _, hunk := range hunks {
		targetsMap[TargetKey(hunk.File, Target{})] = struct{}{}
		if _, ok := rangesMap[hunk.File]; ok {
			rangesMap[hunk.File] = append(rangesMap[hunk.File], hunk.Range)
			continue
		}

		rangesMap[hunk.File] = []Range{hunk.Range}
	}

	// Populate rules map.
	rulesMap := make(map[string][]Rule, len(hunks))
	visited := make(map[string]struct{})

	// First pass parses all the rules from files that are present in the diff.
	for pathname, ranges := range rangesMap {
		visited[pathname] = struct{}{}
		rules, err := RulesFromFile(pathname, ranges, options)
		if err != nil {
			return nil, nil, err
		}

		if len(rules) == 0 {
			continue
		}

		for _, rule := range rules {
			targetsMap[TargetKey(pathname, Target{ID: rule.ID})] = struct{}{}
		}

		rulesMap[pathname] = rules
	}

	// Second pass parses all targets rules that were not present in the diff.
	for _, rules := range rulesMap {
		for _, rule := range rules {
			for _, target := range rule.Targets {
				if target.File == nil {
					continue
				}

				if _, ok := visited[*target.File]; ok {
					continue
				}

				visited[*target.File] = struct{}{}
				moreRules, err := RulesFromFile(*target.File, nil, options)
				if err != nil {
					return nil, nil, err
				}

				if len(moreRules) == 0 {
					continue
				}

				for _, rule := range moreRules {
					targetsMap[TargetKey(*target.File, Target{ID: rule.ID})] = struct{}{}
				}

				rulesMap[*target.File] = moreRules
			}
		}
	}

	return rulesMap, targetsMap, nil
}

// RulesFromFile parses rules from the given file and returns the list of rules.
func RulesFromFile(file string, ranges []Range, options LintOptions) ([]Rule, error) {
	// Parse rules for the file.
	log.Println("parsing rules for file", file)
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %s", file)
	}

	defer f.Close()

	templates, err := options.TemplatesFromFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse templates for file %s", file)
	}

	tokens, err := lex(f, lexOptions{file, templates})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lex file %s", file)
	}

	rules, err := parseRules(file, tokens, ranges)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rules for file %s", file)
	}

	// Add rules to the map.
	return rules, nil
}

//LINT.IF lex.go

// hello

//LINT.END
