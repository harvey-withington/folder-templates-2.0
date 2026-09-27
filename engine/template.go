// Package foldertemplate generates folder structures from templates — a Go
// port of the Folder Templates C# utility (MIT). The on-disk template format
// is unchanged: a template is an ordinary folder containing a `.ft/` config
// directory with a `template.json`, which is never copied to the output.
//
// Port contract (must hold for templates authored by the C# app to keep working):
//   - JSON keys are read case-insensitively and written camelCase.
//   - Parameters with ReplaceInFileNames rename every file/folder (and the
//     template root) via regex Match → value; default Match is the literal
//     `\{Name\}`.
//   - Only files with the extra extension `.ft$` get content processing:
//     `{{$param}}` tokens (case-insensitive) are replaced for parameters with
//     ReplaceInFiles; unknown tokens pass through; the `.ft$` suffix is
//     stripped. All other files are copied byte-for-byte.
//   - Missing parameter values resolve to Value ?? DefaultValue ?? "".
//   - Parameters without a Prompt are internal (never shown in UI).
package foldertemplate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the template config directory, never copied to output.
	ConfigDirName = ".ft"
	// ConfigFileName is the template descriptor inside ConfigDirName.
	ConfigFileName = "template.json"
	// ContentExt marks files whose content gets token replacement; stripped on output.
	ContentExt = ".ft$"
)

// ErrNotATemplate is returned by Load when the folder has no .ft/template.json.
var ErrNotATemplate = errors.New("folder contains no " + ConfigDirName + "/" + ConfigFileName)

// Parameter mirrors the C# app's parameter model. Pointer fields distinguish
// absent/null from empty string so Save round-trips what the C# app wrote.
type Parameter struct {
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	Prompt             *string `json:"prompt"`
	Placeholder        *string `json:"placeholder"`
	DefaultValue       *string `json:"defaultValue"`
	Match              *string `json:"match"`
	ReplaceInFileNames bool    `json:"replaceInFileNames"`
	ReplaceInFiles     bool    `json:"replaceInFiles"`
}

// Internal reports whether the parameter is hidden from UI (no Prompt).
func (p Parameter) Internal() bool {
	return p.Prompt == nil || *p.Prompt == ""
}

// Template is a loaded template: descriptor plus its source folder.
type Template struct {
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	DefaultTargetPath string      `json:"defaultTargetPath"`
	Parameters        []Parameter `json:"parameters"`

	dir string
}

// Dir returns the template's source folder (the folder containing .ft/).
func (t *Template) Dir() string { return t.dir }

// Load reads <dir>/.ft/template.json. JSON keys match case-insensitively
// (encoding/json semantics), accepting both the C# app's historical PascalCase
// and its camelCase output.
func Load(dir string) (*Template, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(abs, ConfigDirName, ConfigFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", dir, ErrNotATemplate)
	}
	if err != nil {
		return nil, err
	}
	t := &Template{dir: abs}
	if err := json.Unmarshal(raw, t); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ConfigFileName, err)
	}
	return t, nil
}

// Save writes the descriptor to <dir>/.ft/template.json in camelCase,
// matching the C# app's serializer. dir may be a new folder; .ft/ is created.
func Save(t *Template, dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(abs, ConfigDirName), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(abs, ConfigDirName, ConfigFileName), raw, 0o644); err != nil {
		return err
	}
	t.dir = abs
	return nil
}
