package foldertemplate

import (
	"fmt"
	"os"
	"path/filepath"
)

// Result reports what Generate produced.
type Result struct {
	// RootPath is the generated root folder (targetParent + renamed root name).
	RootPath string
	// FilesWritten counts regular files created (dirs excluded).
	FilesWritten int
	// Warnings lists non-fatal skips (e.g. symlinks).
	Warnings []string
}

// Generate materializes the template under targetParent.
//
// values holds the user's answers keyed by declared parameter name
// (case-insensitive; missing → DefaultValue → ""). extra holds caller context
// parameters (e.g. BRUV's bruvBrand/bruvDate) resolvable in names and content
// without being declared — declared parameters with the same name win.
//
// The generated root is targetParent/<renamed template root name> and must not
// already exist. targetParent is created if missing. Generating into the
// template folder itself is refused (recursion guard).
func Generate(t *Template, targetParent string, values, extra map[string]string, opts *Options) (*Result, error) {
	o := opts.withDefaults()
	if err := guardTarget(t.dir, targetParent); err != nil {
		return nil, err
	}
	bindings, err := resolveBindings(t, values, extra, o)
	if err != nil {
		return nil, err
	}
	plan, err := buildPlan(t, bindings)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(targetParent, 0o755); err != nil {
		return nil, err
	}
	rootPath := filepath.Join(targetParent, plan.RootName)
	if _, err := os.Stat(rootPath); err == nil {
		return nil, fmt.Errorf("target %q already exists — refusing to generate into it", rootPath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Mkdir(rootPath, 0o755); err != nil {
		return nil, err
	}

	res := &Result{RootPath: rootPath, Warnings: plan.Warnings}
	for _, e := range plan.Entries {
		dst := filepath.Join(rootPath, filepath.FromSlash(e.OutputRel))
		src := filepath.Join(t.dir, filepath.FromSlash(e.SourceRel))
		switch {
		case e.IsDir:
			if err := os.Mkdir(dst, 0o755); err != nil {
				return nil, err
			}
		case e.Process:
			if err := processContentFile(src, dst, bindings, o.ContentSizeLimit); err != nil {
				return nil, err
			}
			res.FilesWritten++
		default:
			if err := copyFile(src, dst); err != nil {
				return nil, err
			}
			res.FilesWritten++
		}
	}
	return res, nil
}
