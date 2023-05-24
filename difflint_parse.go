package difflint

import (
	"bufio"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type token struct {
	directive directive
	args      []string // ["if"] or ["then", "test.go:ID"]

	line int32 // Line number of the token.
}

type directive string

const (
	directiveIf   directive = "IF"
	directiveThen directive = "THEN"
)

type directiveTemplate struct {
	templates []string // "//LINT.?" | "#LINT.?" | "<!-- LINT.? -->"
	fileTypes []string // []string{"js", "ts", "tsx", "jsonc"}
}

type lexOptions struct {
	// File is specifier that is being linted.
	file string

	// Templates is a list of templates that define which directives to parse.
	templates []directiveTemplate
}

// templateFromFileType returns the directive template for the given file type.
func (o *lexOptions) templatesFromFile() (*[]string, error) {
	fileType := strings.TrimPrefix(filepath.Ext(o.file), ".")
	if fileType == "" {
		return nil, errors.Errorf("file %q has no extension", o.file)
	}

	for _, template := range o.templates {
		for _, t := range template.fileTypes {
			if t == fileType {
				return &template.templates, nil
			}
		}
	}

	return nil, errors.Errorf("no directive template found for file type %q", fileType)
}

func lex(r io.Reader, options lexOptions) ([]token, error) {
	templates, err := options.templatesFromFile()
	if err != nil {
		return nil, err
	}

	// tokens is the list of tokens that are found in the file.
	var tokens []token

	// lineCount is the current line number.
	var lineCount int32

	// Read the file line by line.
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Check if the line is a directive.
		token, found, err := parseToken(line, lineCount, *templates)
		if err != nil {
			return nil, err
		}

		if !found {
			continue
		}

		tokens = append(tokens, *token)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

func parseToken(line string, lineNumber int32, templates []string) (*token, bool, error) {
	for _, template := range templates {
		prefix, suffix, found := strings.Cut(template, "?")
		if !found {
			return nil, false, errors.New("template is missing ?")
		}

		if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
			continue
		}

		// Remove the prefix and suffix.
		s := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
		args := strings.Split(s, " ")
		d, err := parseDirective(args[0])
		if err != nil {
			return nil, false, err
		}

		return &token{
			directive: d,
			args:      args[1:],
			line:      lineNumber,
		}, true, nil
	}

	return nil, false, errors.New("line is missing directive")
}

func parseDirective(s string) (directive, error) {
	d := directive(s)
	switch d {
	case directiveIf, directiveThen:
		return d, nil
	default:
		return "", errors.Errorf("unknown directive %q", d)
	}
}

func parseRules(file string, tokens []token) ([]Rule, error) {
	// Current rule being parsed.
	r := Rule{}

	var rules []Rule
	for _, token := range tokens {
		switch token.directive {
		case directiveIf:
			if r.Hunk.File != "" {
				return nil, errors.New("unexpected IF directive")
			}

			r.Hunk.File = file
			r.Hunk.Range = Range{Start: token.line}
			if len(token.args) == 1 {
				r.ID = &(token.args[0])
			}

			if len(token.args) > 1 {
				return nil, errors.Errorf("unexpected arguments %v", token.args)
			}

		case directiveThen:
			if r.Hunk.File == "" {
				return nil, errors.New("unexpected THEN directive")
			}

			targets, err := parseTargets(parseTargetsOptions{
				args:       token.args,
				allowEmpty: r.ID != nil,
			})
			if err != nil {
				return nil, err
			}

			r.Targets = targets
			r.Hunk.Range.End = token.line
			rules = append(rules, r)

			// Reset the rule.
			r = Rule{}

		default:
			return nil, errors.Errorf("unknown directive %q", token.directive)
		}
	}

	return rules, nil
}

type parseTargetsOptions struct {
	args       []string
	allowEmpty bool
}

func parseTargets(o parseTargetsOptions) ([]Target, error) {
	if !o.allowEmpty && len(o.args) == 0 {
		return nil, errors.New("missing target")
	}

	var targets []Target
	for _, arg := range o.args {
		file, id, hasID := strings.Cut(arg, ":")
		target := Target{File: file}
		if hasID {
			target.ID = &id
		}

		targets = append(targets, target)
	}

	return targets, nil
}
