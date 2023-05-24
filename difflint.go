package difflint

import (
	"fmt"
	"io"
	"log"

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

// LintOptions represents the options for a linting operation.
type LintOptions struct {
	// Reader is the reader from which the diff is read.
	Reader io.Reader

	// Include is a list of file patterns to include in the linting.
	Include []string

	// Exclude is a list of file patterns to exclude from the linting.
	Exclude []string
}

//LINT.IF ./lex.go:id00

// Between returns true if changes exist between the given line numbers.
func Between(a, b Range) bool {
	return a.Start <= b.Start && a.End >= b.End
}

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
	rulesMap, err := RulesFromHunks(hunks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rules from hunks")
	}

	// Print out rules.
	log.Println("Rules:", rulesMap)

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
				if rule.ID == target.ID && Between(hunk.Range, rule.Hunk.Range) {
					return true
				}
			}
		}

		if hunk.File == target.File {
			return true
		}
	}

	return false
}

// Entrypoint for the difflint command.
func Do(r io.Reader, include, exclude []string) error {
	// Lint the hunks.
	result, err := Lint(LintOptions{
		Reader:  r,
		Include: include,
		Exclude: exclude,
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
