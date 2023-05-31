package difflint

import (
	"encoding/json"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/sourcegraph/go-diff/diff"
)

// Hello world

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

//LINT.END :id01

// Hunk represents a diff hunk that must be present in the diff.
type Hunk struct {
	// File specifier of the defined range.
	File string

	// Range of code in which a diff hunk intersects.
	Range Range
}

// UnsatisfiedRule represents a rule that is not satisfied.
type UnsatisfiedRule struct {
	// Rule that is not satisfied.
	Rule

	// UnsatisfiedTargets is the list of target indices that are not satisfied.
	UnsatisfiedTargets []int32
}

// Result of a linting operation.
type LintResult struct {
	// List of rules that were not satisfied.
	UnsatisfiedRules []UnsatisfiedRule
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

	// Analyze the rules.
	if errs := Analyze(rulesMap); len(errs) > 0 {
		return nil, errors.Errorf("analysis errors: %v", errs)
	}

	// Collect the rules that are not satisfied.
	unsatisfiedRules, err := Check(rulesMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check rules")
	}

	return &LintResult{UnsatisfiedRules: unsatisfiedRules}, nil
}

// TargetKey returns the key for the given target.
// TODO: Make this more consistent so that the Analyze function can work.
func TargetKey(pathname string, target Target) string {
	targetKey := pathname
	if target.File != nil && *target.File != "" {
		targetKey = *target.File
		if !filepath.IsAbs(targetKey) {
			targetKey = filepath.Join(filepath.Dir(pathname), targetKey)
		}
	}

	if target.ID != nil {
		targetKey += ":" + *target.ID
	}

	return targetKey
}

// Analyze analyzes the rules and returns a list of errors.
func Analyze(rulesMap map[string][]Rule) []error {
	// Append an error for each rule that has a non-existent target.
	var errs []error

	// Construct a set of all the targets that exist.
	targets := make(map[string]struct{})
	for pathname, rules := range rulesMap {
		for _, rule := range rules {
			targets[pathname] = struct{}{}
			if rule.ID == nil || *rule.ID == "" {
				continue
			}

			targetKey := TargetKey(pathname, Target{ID: rule.ID})
			println("targetKey1", targetKey)
			targets[targetKey] = struct{}{}
		}
	}

	// Iterate through the entire map of rules again to find all the rules with targets that don't exist.
	for pathname, rules := range rulesMap {
		for _, rule := range rules {
			for _, target := range rule.Targets {
				targetKey := TargetKey(pathname, target)
				if _, ok := targets[targetKey]; !ok {
					println("targetKey2", targetKey)
					errs = append(errs, errors.Errorf("rule %s:L%d-L%d has non-existent target %q", pathname, rule.Hunk.Range.Start, rule.Hunk.Range.End, targetKey))
				}
			}
		}
	}

	return errs
}

// Check returns the list of unsatisfied rules for the given map of rules.
func Check(rulesMap map[string][]Rule) ([]UnsatisfiedRule, error) {
	var unsatisfiedRules []UnsatisfiedRule
	for pathnameA, rulesA := range rulesMap {
	outer:
		for i, ruleA := range rulesA {
			// Skip if ruleA is not present or if it has no targets.
			if len(ruleA.Targets) == 0 || !ruleA.Present {
				continue
			}

			for pathnameB, rulesB := range rulesMap {
			inner:
				for j, ruleB := range rulesB {
					// Skip if both rules are present or if ruleA is the same as ruleB.
					if ruleB.Present || (pathnameA == pathnameB && i == j) {
						continue
					}

					// Given that ruleA is present and ruleB is not present, check if ruleA
					// is satisfied by ruleB.
					unsatisfiedTargetIndices := make([]int32, 0, len(ruleA.Targets))
					for k, target := range ruleA.Targets {
						// ruleA is satisfied by ruleB if ruleB matches a target of ruleA.
						satisfied := target.ID == ruleB.ID && ((target.File == nil && pathnameA == pathnameB) || (*target.File == pathnameB))
						if satisfied {
							continue inner
						}

						unsatisfiedTargetIndices = append(unsatisfiedTargetIndices, int32(k))
					}

					unsatisfiedRules = append(unsatisfiedRules, UnsatisfiedRule{
						Rule:               ruleA,
						UnsatisfiedTargets: unsatisfiedTargetIndices,
					})
					continue outer
				}
			}
		}
	}

	return unsatisfiedRules, nil
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
			"py":       {0},
			"sh":       {0},
			"go":       {1},
			"js":       {1, 2},
			"jsx":      {1, 2},
			"mjs":      {1, 2},
			"ts":       {1, 2},
			"tsx":      {1, 2},
			"jsonc":    {1, 2},
			"c":        {1, 2},
			"cc":       {1, 2},
			"cpp":      {1, 2},
			"h":        {1, 2},
			"hpp":      {1, 2},
			"java":     {1},
			"rs":       {1},
			"swift":    {1},
			"svelte":   {1, 2, 3},
			"css":      {2},
			"html":     {3},
			"md":       {3},
			"markdown": {3},
			"bas":      {4},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to lint hunks")
	}

	// If there are no unsatisfied rules, return nil.
	if len(result.UnsatisfiedRules) == 0 {
		return nil
	}

	// Print the unsatisfied rules.
	for _, rule := range result.UnsatisfiedRules {
		// Print the unsatisfied rule.
		output, err := json.MarshalIndent(rule, "", "  ")
		if err != nil {
			return errors.Wrap(err, "failed to marshal result")
		}
		log.Println("Unsatisfied rule:", string(output))
	}

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
