package difflint

import (
	"encoding/json"
	"log"
)

var (
	// DefaultTemplates is the default list of directive templates.
	DefaultTemplates = []string{
		"#LINT.?",
		"//LINT.?",
		"/*LINT.?",
		"<!--LINT.?",
		"'LINT.?",
	}

	// DefaultFileExtMap is the default map of file extensions to directive templates.
	DefaultFileExtMap = map[string][]int{
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
	}
)

// ExtFileJSON is a JSON representation of a file extension to directive template map.
type ExtFileJSON map[string][]string

// ExtMap represents the extensions and templates for a linting operation.
type ExtMap struct {
	// Templates is the list of directive templates.
	Templates []string

	// FileExtMap is a map of file extensions to directive templates.
	FileExtMap map[string][]int
}

// NewExtMap returns a new ExtMap instance.
func NewExtMap(path *string) *ExtMap {
	o := &ExtMap{
		Templates:  DefaultTemplates,
		FileExtMap: DefaultFileExtMap,
	}

	// If a path is provided, update the templates and file extension map.
	if path != nil {
		// Unmarshal the JSON file.
		var extFile ExtFileJSON
		if err := json.Unmarshal([]byte(*path), &extFile); err != nil {
			log.Fatalf("error unmarshaling JSON file %q: %v", *path, err)
		}

		// Update the templates and file extension map.
		for ext, tpls := range extFile {
			for _, tpl := range tpls {
				o.With(ext, tpl)
			}
		}
	}

	return o
}

// With adds a directive template for a file extension.
func (o *ExtMap) With(ext, tpl string) *ExtMap {
	tplIndex := -1
	for i, t := range o.Templates {
		if t == tpl {
			tplIndex = i
			break
		}
	}

	if tplIndex == -1 {
		tplIndex = len(o.Templates)
		o.Templates = append(o.Templates, tpl)
	}

	o.FileExtMap[ext] = append(o.FileExtMap[ext], tplIndex)
	return o
}
