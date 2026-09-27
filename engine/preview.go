package foldertemplate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PreviewEntry is one would-be output item from a dry run.
type PreviewEntry struct {
	SourceRel string `json:"sourceRel"` // "" for the root folder itself
	OutputRel string `json:"outputRel"` // renamed path relative to targetParent
	IsDir     bool   `json:"isDir"`
	Processed bool   `json:"processed"` // content token replacement applied
}

// Preview runs the generator's planning phase in-memory: the resulting tree
// (renamed, .ft$-stripped) with nothing written to disk. Same inputs as
// Generate; same errors (bad patterns, timeouts, case collisions) — so the
// template editor's dry run catches exactly what a real run would.
func Preview(t *Template, values, extra map[string]string, opts *Options) ([]PreviewEntry, []string, error) {
	o := opts.withDefaults()
	bindings, err := resolveBindings(t, values, extra, o)
	if err != nil {
		return nil, nil, err
	}
	plan, err := buildPlan(t, bindings)
	if err != nil {
		return nil, nil, err
	}
	out := make([]PreviewEntry, 0, len(plan.Entries)+1)
	out = append(out, PreviewEntry{SourceRel: "", OutputRel: plan.RootName, IsDir: true})
	for _, e := range plan.Entries {
		out = append(out, PreviewEntry{
			SourceRel: e.SourceRel,
			OutputRel: plan.RootName + "/" + e.OutputRel,
			IsDir:     e.IsDir,
			Processed: e.Process,
		})
	}
	return out, plan.Warnings, nil
}

// RenderFile returns the before/after content of one .ft$ file for the
// editor's dry-run diff view. sourceRel is the path relative to the template
// dir, as reported by Preview. The size ceiling applies.
func RenderFile(t *Template, sourceRel string, values, extra map[string]string, opts *Options) (before, after string, err error) {
	o := opts.withDefaults()
	if !strings.HasSuffix(sourceRel, ContentExt) {
		return "", "", fmt.Errorf("%s is not a %s file", sourceRel, ContentExt)
	}
	bindings, err := resolveBindings(t, values, extra, o)
	if err != nil {
		return "", "", err
	}
	src := filepath.Join(t.dir, filepath.FromSlash(sourceRel))
	info, err := os.Stat(src)
	if err != nil {
		return "", "", err
	}
	if info.Size() > o.ContentSizeLimit {
		return "", "", fmt.Errorf("%s (%d bytes > %d): %w", sourceRel, info.Size(), o.ContentSizeLimit, ErrContentTooLarge)
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return "", "", err
	}
	var rendered strings.Builder
	if err := processContent(strings.NewReader(string(raw)), &rendered, bindings); err != nil {
		return "", "", fmt.Errorf("%s: %w", sourceRel, err)
	}
	return string(raw), rendered.String(), nil
}
