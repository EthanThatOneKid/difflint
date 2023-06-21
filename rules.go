package difflint

import (
	"log"
	"os"
	"sync"

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
	var wg sync.WaitGroup
	for pathname, ranges := range rangesMap {
		rules, err := RulesFromFile(pathname, ranges, visited, &wg, options)
		if err != nil {
			return nil, nil, err
		}

		for _, rule := range rules {
			targetsMap[TargetKey(pathname, Target{ID: rule.ID})] = struct{}{}
		}

		rulesMap[pathname] = rules
	}

	wg.Wait()

	return rulesMap, targetsMap, nil
}

func RulesFromFile(pathname string, ranges []Range, visited map[string]struct{}, wg *sync.WaitGroup, options LintOptions) ([]Rule, error) {
	visited[pathname] = struct{}{}

	// Parse rules for the file.
	log.Println("Parsing rules for file", pathname)
	file, err := os.Open(pathname)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %s", pathname)
	}

	defer file.Close()

	templates, err := options.TemplatesFromFile(pathname)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse templates for file %s", pathname)
	}

	tokens, err := lex(file, lexOptions{
		file:      pathname,
		templates: templates,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lex file %s", pathname)
	}

	rules, err := parseRules(pathname, tokens, ranges)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rules for file %s", pathname)
	}

	for _, rule := range rules {
		for _, target := range rule.Targets {
			if target.File == nil {
				continue
			}

			wg.Add(1)
			go func(pathname string) {
				defer wg.Done()

				if _, ok := visited[pathname]; ok {
					return
				}

				moreRules, err := RulesFromFile(pathname, nil, visited, wg, options)
				if err != nil {
					return
				}

				rules = append(rules, moreRules...)
			}(*target.File)
		}
	}

	return rules, nil
}

//LINT.IF

//LINT.END id
