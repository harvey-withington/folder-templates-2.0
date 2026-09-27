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
	// Exists is set by PreviewAt when the output path is already on disk.
	Exists bool `json:"exists,omitempty"`
}

// PreviewResult is PreviewAt's dry run against a concrete target folder.
type PreviewResult struct {
	RootName   string         `json:"rootName"`
	RootPath   string         `json:"rootPath"`
	RootExists bool           `json:"rootExists"`
	Entries    []PreviewEntry `json:"entries"`
	Warnings   []string       `json:"warnings"`
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

// PreviewAt is Preview against a real target parent: it also applies the
// recursion guard and marks entries (and the root) that already exist, so a
// UI can show conflicts before the user picks a conflict policy.
func PreviewAt(t *Template, targetParent string, values, extra map[string]string, opts *Options) (*PreviewResult, error) {
	if err := guardTarget(t.dir, targetParent); err != nil {
		return nil, err
	}
	entries, warnings, err := Preview(t, values, extra, opts)
	if err != nil {
		return nil, err
	}
	res := &PreviewResult{
		RootName: entries[0].OutputRel,
		RootPath: filepath.Join(targetParent, entries[0].OutputRel),
		Entries:  entries,
		Warnings: warnings,
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	if _, err := os.Stat(res.RootPath); err != nil {
		return res, nil // nothing below a missing root can exist
	}
	res.RootExists = true
	for i := range res.Entries {
		p := filepath.Join(targetParent, filepath.FromSlash(res.Entries[i].OutputRel))
		if _, err := os.Stat(p); err == nil {
			res.Entries[i].Exists = true
		}
	}
	return res, nil
}

// ResolveDefaultTarget returns the folder a template generates into when the
// caller names none — the C# app's rule: DefaultTargetPath if set (a relative
// path, ../ allowed, resolves against the template folder's parent; an
// absolute path is taken as-is), otherwise the template folder's parent.
func ResolveDefaultTarget(t *Template) string {
	parent := filepath.Dir(t.dir)
	dtp := strings.TrimSpace(t.DefaultTargetPath)
	if dtp == "" {
		return parent
	}
	p := filepath.FromSlash(dtp)
	if !filepath.IsAbs(p) {
		p = filepath.Join(parent, p)
	}
	return filepath.Clean(p)
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
