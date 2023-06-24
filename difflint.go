package difflint

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/sourcegraph/go-diff/diff"
)

// Range represents a range of line numbers.
type Range struct {
	// Start line number.
	Start int

	// End line number.
	End int
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
	FileExtMap map[string][]int // map[string][]int{".go": []int{0}, ".py": []int{1}}

	// DefaultTemplate is the default directive template.
	DefaultTemplate int
}

// TemplatesFromFile returns the directive templates for the given file type.
func (o *LintOptions) TemplatesFromFile(file string) ([]string, error) {
	fileType := strings.TrimPrefix(filepath.Ext(file), ".")
	templateIndices, ok := o.FileExtMap[fileType]
	if !ok {
		templateIndices = []int{o.DefaultTemplate}
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
	UnsatisfiedTargets map[int]struct{}
}

// UnsatisfiedRules is a list of unsatisfied rules.
type UnsatisfiedRules []UnsatisfiedRule

// String returns a string representation of the unsatisfied rules.
func (r *UnsatisfiedRules) String() string {
	var b strings.Builder
	for _, rule := range *r {
		b.WriteString("rule (")
		b.WriteString(rule.Rule.Hunk.File)
		b.WriteString(":")
		b.WriteString(fmt.Sprintf("%d", rule.Rule.Hunk.Range.Start))
		b.WriteString(",")
		b.WriteString(rule.Rule.Hunk.File)
		b.WriteString(":")
		b.WriteString(fmt.Sprintf("%d", rule.Rule.Hunk.Range.End))
		b.WriteString(") not satisfied for targets:\n")
		for i, target := range rule.Targets {
			if _, ok := rule.UnsatisfiedTargets[i]; !ok {
				continue
			}

			key := TargetKey(rule.Rule.Hunk.File, target)
			b.WriteString("  ")
			b.WriteString(key)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Result of a linting operation.
type LintResult struct {
	// List of rules that were not satisfied.
	UnsatisfiedRules UnsatisfiedRules
}

// Walk walks the file tree rooted at root, calling callback for each file or
// directory in the tree, including root.
func Walk(root string, include []string, exclude []string, callback filepath.WalkFunc) error {
	isHidden := func(path string) bool {
		components := strings.Split(path, string(os.PathSeparator))
		for _, component := range components {
			if strings.HasPrefix(component, ".") && component != "." && component != ".." {
				return true
			}
		}
		return false
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if isHidden(path) {
			return nil
		}

		included, err := Include(path, include, exclude)
		if err != nil {
			return err
		}

		if !included {
			return nil
		}

		return callback(path, info, nil)
	})
}

// Lint lints the given hunks against the given rules and returns the result.
func Lint(o LintOptions) (*LintResult, error) {
	// Parse the diff hunks.
	hunks, err := ParseHunks(o.Reader, o.Include, o.Exclude)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse diff hunks")
	}

	// Parse rules from hunks.
	rulesMap, presentTargetsMap, err := RulesMapFromHunks(hunks, o)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rules from hunks")
	}

	// Collect the rules that are not satisfied.
	unsatisfiedRules, err := Check(rulesMap, presentTargetsMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check rules")
	}

	// Filter out rules that are not intended to be included in the output.
	var filteredUnsatisfiedRules UnsatisfiedRules
	for _, rule := range unsatisfiedRules {
		included, err := Include(rule.Rule.Hunk.File, o.Include, o.Exclude)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if file is included")
		}

		if !included {
			continue
		}

		filteredUnsatisfiedRules = append(filteredUnsatisfiedRules, rule)
	}

	return &LintResult{UnsatisfiedRules: filteredUnsatisfiedRules}, nil
}

// TargetKey returns the key for the given target.
func TargetKey(pathname string, target Target) string {
	key := string(pathname)
	if target.File != nil && *target.File != "" {
		key = *target.File
		if isRelativeToCurrentDirectory(*target.File) {
			key = filepath.Join(filepath.Dir(pathname), *target.File)
		}
	}

	if target.ID != nil {
		key += ":" + *target.ID
	}

	return filepath.Clean(key)
}

// isRelativeToCurrentDirectory returns true if the given path is a specific relative path.
// A specific relative path implies that the user specifically intends to target a
// path relative to the current directory.
func isRelativeToCurrentDirectory(path string) bool {
	// Check if the path is a relative path
	if !strings.HasPrefix(path, "/") {
		// Check if the path starts with "./" or "../"
		return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
	}

	return false
}

// Check returns the list of unsatisfied rules for the given map of rules.
func Check(rulesMap map[string][]Rule, targetsMap map[string]struct{}) (UnsatisfiedRules, error) {
	var unsatisfiedRules UnsatisfiedRules

	// Check each rule.
	for _, rules := range rulesMap {
		for _, rule := range rules {
			if rule.Present {
				continue
			}

			unsatisfiedTargets := make(map[int]struct{}, len(rule.Targets))
			for i, target := range rule.Targets {
				key := TargetKey(rule.Hunk.File, target)
				if _, ok := targetsMap[key]; ok {
					unsatisfiedTargets[i] = struct{}{}
				}
			}

			// If there are unsatisfied targets, then the rule is unsatisfied.
			if len(unsatisfiedTargets) > 0 {
				unsatisfiedRules = append(unsatisfiedRules, UnsatisfiedRule{
					Rule:               rule,
					UnsatisfiedTargets: unsatisfiedTargets,
				})
			}
		}
	}

	// Return the unordered list of unsatisfied rules.
	return unsatisfiedRules, nil
}

// Do is the difflint command's entrypoint.
func Do(r io.Reader, include, exclude []string, extMapPath string) (UnsatisfiedRules, error) {
	// Parse options.
	extMap := NewExtMap(extMapPath)

	// Lint the hunks.
	result, err := Lint(LintOptions{
		Reader:          r,
		Include:         include,
		Exclude:         exclude,
		DefaultTemplate: 0,
		Templates:       extMap.Templates,
		FileExtMap:      extMap.FileExtMap,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to lint hunks")
	}

	return result.UnsatisfiedRules, nil
}

// ParseHunks parses the input diff and returns the extracted file paths along
// with associated line number ranges.
func ParseHunks(r io.Reader, include, exclude []string) ([]Hunk, error) {
	diffs, err := diff.NewMultiFileDiffReader(r).ReadAllFiles()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read files")
	}

	hunks := make([]Hunk, 0, len(diffs))
	for _, d := range diffs {
		for _, h := range d.Hunks {
			hunk := Hunk{
				File: strings.TrimPrefix(d.NewName, "b/"),
				Range: Range{
					Start: int(h.NewStartLine),
					End:   int(h.NewStartLine + h.NewLines - 1),
				}}
			hunks = append(hunks, hunk)
		}
	}

	return hunks, nil
}

// Include determines if a given diff should be included in the linting process.
func Include(pathname string, include, exclude []string) (bool, error) {
	// If there are no include or exclude rules, return true.
	if len(include) == 0 && len(exclude) == 0 {
		return true, nil
	}

	// If there are exclude rules, check if the diff matches any of them.
	if len(exclude) > 0 {
		for _, e := range exclude {
			if matched, err := filepath.Match(e, pathname); err != nil {
				return false, errors.Wrap(err, "failed to match exclude rule")
			} else if matched {
				return false, nil
			}
		}
	}

	// If there are include rules, check if the diff matches any of them.
	if len(include) > 0 {
		for _, i := range include {
			if matched, err := filepath.Match(i, pathname); err != nil {
				return false, errors.Wrap(err, "failed to match include rule")
			} else if matched {
				return true, nil
			}
		}
	}

	return false, nil
}
