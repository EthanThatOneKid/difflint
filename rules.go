package difflint

import (
	"fmt"
	"log"
	"os"
	"strings"

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

	// ID is an optional, unique identifier for the rule.
	ID *string
}

// RulesMapFromHunks parses rules from the given hunks by file name and
// returns the map of rules.
func RulesMapFromHunks(hunks []Hunk, options LintOptions) (map[string][]Rule, error) {
	rulesMap := make(map[string][]Rule, len(hunks))
	for _, hunk := range hunks {
		if _, ok := rulesMap[hunk.File]; ok {
			continue
		}

		// Parse rules for the file.
		log.Println("Parsing rules for file", hunk.File)
		file, err := ReadB(hunk.File)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to open file %s", hunk.File)
		}
		defer file.Close()

		templates, err := options.TemplatesFromFile(hunk.File)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse templates for file %s", hunk.File)
		}

		tokens, err := lex(file, lexOptions{
			file:      hunk.File,
			templates: templates,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to lex file %s", hunk.File)
		}

		rules, err := parseRules(hunk.File, tokens)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rules for file %s", hunk.File)
		}

		rulesMap[hunk.File] = rules
	}

	return rulesMap, nil
}

// ReadA reads the file with the given name from the a/ directory.
func ReadA(oldName string) (*os.File, error) {
	return ReadFile(oldName, "a/")
}

// ReadB reads the file with the given name from the b/ directory.
func ReadB(newName string) (*os.File, error) {
	return ReadFile(newName, "b/")
}

// ReadFile reads the file with the given name from the given prefix.
func ReadFile(prefixedFile, prefix string) (*os.File, error) {
	if !strings.HasPrefix(prefixedFile, prefix) {
		return nil, fmt.Errorf("unexpected file name: %s", prefixedFile)
	}

	relativePath := strings.TrimPrefix(prefixedFile, prefix)
	f, err := os.Open(relativePath)
	if err != nil {
		return nil, fmt.Errorf("unexpected to open file %s: %v", prefixedFile, err)
	}

	return f, nil
}
