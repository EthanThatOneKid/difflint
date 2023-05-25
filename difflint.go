package difflint

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/sourcegraph/go-diff/diff"
)

// Range represents a range of line numbers.
type Range struct {
	// Start line number.
	Start int32

	// End line number.
	End int32
}

// Intersects returns true if the given ranges intersect.
func Intersects(a, b Range) bool {
	return a.Start <= b.End && b.Start <= a.End
}

// LintOptions represents the options for a linting operation.
type LintOptions struct {
	// Reader is the reader from which the diff is read.
	Reader io.Reader

	// Include is a list of file patterns to include in the linting.
	Include []string

	// Exclude is a list of file patterns to exclude from the linting.
	Exclude []string

	// Templates is the list of directive templates.
	Templates []string // []string{"//LINT.?", "#LINT.?", "<!-- LINT.? -->"}

	// FileExtMap is a map of file extensions to directive templates.
	FileExtMap map[string][]int32 // map[string][]int32{".go": []int32{0}, ".py": []int32{1}}

	// DefaultTemplate is the default directive template.
	DefaultTemplate int32
}

// TemplatesFromFile returns the directive templates for the given file type.
func (o *LintOptions) TemplatesFromFile(file string) ([]string, error) {
	fileType := strings.TrimPrefix(filepath.Ext(file), ".")
	if fileType == "" {
		return nil, errors.Errorf("file %q has no extension", file)
	}

	templateIndices, ok := o.FileExtMap[fileType]
	if !ok {
		return nil, errors.Errorf("no directive template found for file type %q", fileType)
	}

	var filteredTemplates []string
	for _, i := range templateIndices {
		filteredTemplates = append(filteredTemplates, o.Templates[i])
	}

	if len(filteredTemplates) == 0 {
		filteredTemplates = append(filteredTemplates, o.Templates[o.DefaultTemplate])
	}
	return filteredTemplates, nil
}

//LINT.IF ./lex.go:id00

//LINT.END id01

// Hunk represents a diff hunk that must be present in the diff.
type Hunk struct {
	// File specifier of the defined range.
	File string

	// Range of code in which a diff hunk intersects.
	Range Range
}

// Result of a linting operation.
type LintResult struct {
	// List of rules that were not satisfied.
	UnsatisfiedRules []Rule
}

// Lint lints the given hunks against the given rules and returns the result.
func Lint(o LintOptions) (*LintResult, error) {
	// Parse the diff hunks.
	hunks, err := ParseHunks(o.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse diff hunks")
	}

	// Parse rules from hunks.
	rulesMap, err := RulesMapFromHunks(hunks, o)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rules from hunks")
	}

	// TODO: Remove this.
	payload, err := json.MarshalIndent(rulesMap, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rules")
	}
	log.Println("Rules:", string(payload))

	return nil /* TODO */, nil
	// Check if the diff contains any hunks that are covered by a rule.
	var unsatisfiedRules []Rule
	for rulePath, rules := range rulesMap {
		for _, rule := range rules {
			// Check if the rule is in its own file.
			if rule.Hunk.File == rulePath {
				log.Println("Warning: Rule in own file does nothing:", rule)
			}

			// Check if the diff contains all the targets of the rule.
			for _, target := range rule.Targets {
				// Check if the target is a file or a range of code.
				if !Check(target, hunks, rules) {
					continue
				}

				unsatisfiedRules = append(unsatisfiedRules, rule)
			}
		}
	}

	return &LintResult{
		UnsatisfiedRules: unsatisfiedRules,
	}, nil
}

// Check checks if the given hunk is covered by the given target.
func Check(target Target, hunks []Hunk, rules []Rule) bool {
	for _, hunk := range hunks {
		if target.ID != nil {
			for _, rule := range rules {
				if rule.ID == target.ID && Intersects(hunk.Range, rule.Hunk.Range) {
					return true
				}
			}
		}

		if hunk.File == *target.File {
			return true
		}
	}

	return false
}

// Entrypoint for the difflint command.
func Do(r io.Reader, include, exclude []string) error {
	// Lint the hunks.
	result, err := Lint(LintOptions{
		Reader:          r,
		Include:         include,
		Exclude:         exclude,
		DefaultTemplate: 0,
		Templates: []string{
			"#LINT.?",
			"//LINT.?",
			"/*LINT.?",
			"<!--LINT.?",
			"'LINT.?",
		},
		FileExtMap: map[string][]int32{
			"py":     {0},
			"go":     {1},
			"js":     {1, 2},
			"jsx":    {1, 2},
			"mjs":    {1, 2},
			"ts":     {1, 2},
			"tsx":    {1, 2},
			"jsonc":  {1, 2},
			"c":      {1, 2},
			"cc":     {1, 2},
			"cpp":    {1, 2},
			"h":      {1, 2},
			"hpp":    {1, 2},
			"java":   {1},
			"rs":     {1},
			"swift":  {1},
			"svelte": {1, 2, 3},
			"css":    {2},
			"html":   {3},
			"md":     {3},
			"bas":    {4},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to lint hunks")
	}

	// Print the result.
	fmt.Println(result)

	return nil
}

// ParseHunks parses the input diff and returns the extracted file paths along
// with associated line number ranges.
func ParseHunks(r io.Reader) ([]Hunk, error) {
	diffs, err := diff.NewMultiFileDiffReader(r).ReadAllFiles()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read files")
	}

	hunks := make([]Hunk, 0, len(diffs))
	for _, d := range diffs {
		for _, h := range d.Hunks {
			hunk := Hunk{
				File: d.NewName,
				Range: Range{
					Start: h.NewStartLine,
					End:   h.NewStartLine + h.NewLines - 1,
				}}
			hunks = append(hunks, hunk)
		}
	}

	return hunks, nil
}
