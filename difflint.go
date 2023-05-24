package difflint

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/sourcegraph/go-diff/diff"
)

// Range represents a range of line numbers.
type Range struct {
	Start int32 // Start line number
	End   int32 // End line number
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

// .js, .ts, .tsx, .jsonc -> //LINT.

//LINT.THEN ./test.go ./test2.go:ID

//LINT.ID id

// Target represents a file or range of code that must be present in the diff if a diff hunk is present.
type Target struct {
	// File specifier expected to contain a diff hunk.
	File string

	// ID is the ID of the range of code in which a diff hunk intersects.
	ID *string
	// Range *Range
}

//LINT.IF id

//LINT.THEN ./test.go:testID

//LINT.ID
//LINT.IF [id]
//LINT.THEN [file:ID]...

// A rule says that file or range of code must be present in the diff if another range is present.
type Rule struct {
	// Hunk is the diff hunk that must be present in the diff.
	Hunk Hunk

	// Targets are the files or ranges of code that must be present in the diff if the hunk is present.
	Targets []Target

	// ID is an optional, unique identifier for the rule.
	ID *string
}

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

	// First step is to load all relevant rules.
	var fs map[string][]Rule
	for _, hunk := range hunks {
		if _, ok := fs[hunk.File]; ok {
			continue
		}
		
		// Parse rules for the file.
			log.Println("Parsing rules for file", hunk.File)
			fs[hunk.File] = []Rule{
				Hunk: hunk,
				Targets: []Target{hunk}
			}
		
		// fs[hunk.File] = append(fs[hunk.File], Rule{
		// 	Hunk: hunk,
		// })
	}

	// Next step is to check if the diff contains any hunks that are covered by a rule.
	// tokens, err := lex(o.Reader, lexOptions{
	// 	templates: []directiveTemplate{
	// Next step is to check if the diff contains any hunks that are covered by a rule.

	// Next step is to check if the diff contains all the targets of the rules that are covered by a hunk.

	return nil, nil
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

	// Print the result
	fmt.Println(result)

	return nil
}

// Given what has changed in the diff, we need to figure out which checks to run.
// Given what has changed in the diff that requires

// DiffRange represents a range of line numbers in a diff.
// type DiffRange struct {
// 	Range   Range  // Line number range
// 	NewName string // Name of the file in the diff
// }

// Between returns true if changes exist between the given line numbers.
// func Between(d diff.FileDiff, start, end int32) bool {
// 	for _, h := range d.Hunks {
// 		newEndLine := h.NewStartLine + h.NewLines - 1
// 		if h.NewStartLine <= start && newEndLine >= end {
// 			return true
// 		}
// 	}

// 	return false
// }

// TODO: Implement this function
// func Lint() // Lint passes options to the linter and returns the result.

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

func ReadA(oldName string) (*os.File, error) {
	return ReadFile(oldName, "a/")
}

func ReadB(newName string) (*os.File, error) {
	return ReadFile(newName, "b/")
}

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
