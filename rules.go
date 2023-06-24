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
// returns the map of rules and the set of all the target keys that are present.
func RulesMapFromHunks(hunks []Hunk, options LintOptions) (map[string][]Rule, map[string]struct{}, error) {
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

	rulesMap := make(map[string][]Rule, len(hunks))
	err := Walk(".", nil, nil, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", file)
		}
		defer f.Close()

		templates, err := options.TemplatesFromFile(file)
		if err != nil {
			return errors.Wrapf(err, "failed to parse templates for file %s", file)
		}

		tokens, err := lex(f, lexOptions{file, templates})
		if err != nil {
			return errors.Wrapf(err, "failed to lex file %s", file)
		}

		rules, err := parseRules(file, tokens, rangesMap[file])
		if err != nil {
			return errors.Wrapf(err, "failed to parse rules for file %s", file)
		}
		log.Printf("parsed %d rules for file %s", len(rules), file)

		for _, rule := range rules {
			if rule.Hunk.File != file {
				continue
			}

			ranges, ok := rangesMap[file]
			if !ok {
				continue
			}

			for _, rng := range ranges {
				if !Intersects(rule.Hunk.Range, rng) {
					continue
				}

				key := TargetKey(file, Target{
					File: &rule.Hunk.File,
					ID:   rule.ID,
				})
				targetsMap[key] = struct{}{}
			}
		}

		if len(rules) > 0 {
			rulesMap[file] = rules
		}

		return nil
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk files")
	}

	return rulesMap, targetsMap, nil
}
