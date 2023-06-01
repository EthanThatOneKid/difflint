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
func RulesMapFromHunks(hunks []Hunk, options LintOptions) (map[string][]Rule, error) {
	// Separate hunks by file name.
	rangesMap := make(map[string][]Range, len(hunks))
	for _, hunk := range hunks {
		if _, ok := rangesMap[hunk.File]; ok {
			rangesMap[hunk.File] = append(rangesMap[hunk.File], hunk.Range)
			continue
		}

		rangesMap[hunk.File] = []Range{hunk.Range}
	}

	// Populate rules map.
	rulesMap := make(map[string][]Rule, len(hunks))
	for filepath, ranges := range rangesMap {
		// Parse rules for the file.
		log.Println("Parsing rules for file", filepath)
		file, err := os.Open(filepath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to open file %s", filepath)
		}

		defer file.Close()

		templates, err := options.TemplatesFromFile(filepath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse templates for file %s", filepath)
		}

		tokens, err := lex(file, lexOptions{
			file:      filepath,
			templates: templates,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to lex file %s", filepath)
		}

		rules, err := parseRules(filepath, tokens, ranges)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rules for file %s", filepath)
		}

		rulesMap[filepath] = rules
	}

	return rulesMap, nil
}

// // ReadA reads the file with the given name from the a/ directory.
// func ReadA(oldName string) (*os.File, error) {
// 	return ReadFile(oldName, "a/")
// }

// // ReadB reads the file with the given name from the b/ directory.
// func ReadB(newName string) (*os.File, error) {
// 	return ReadFile(newName, "b/")
// }

// // ReadFile reads the file with the given name from the given prefix.
// func ReadFile(prefixedFile, prefix string) (*os.File, error) {
// 	if !strings.HasPrefix(prefixedFile, prefix) {
// 		return nil, fmt.Errorf("unexpected file name: %s", prefixedFile)
// 	}

// 	relativePath := strings.TrimPrefix(prefixedFile, prefix)
// 	f, err := os.Open(relativePath)
// 	if err != nil {
// 		return nil, fmt.Errorf("unexpected to open file %s: %v", prefixedFile, err)
// 	}

// 	return f, nil
// }
