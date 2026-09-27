package foldertemplate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrTargetExists is returned (wrapped) when the output root already exists
// and the conflict policy is ConflictRefuse.
var ErrTargetExists = errors.New("target already exists")

// Result reports what Generate produced.
type Result struct {
	// RootPath is the generated root folder (targetParent + renamed root name).
	RootPath string `json:"rootPath"`
	// FilesWritten counts regular files created or replaced (dirs excluded).
	FilesWritten int `json:"filesWritten"`
	// Skipped lists output-relative files left untouched because they already
	// existed (ConflictMerge only).
	Skipped []string `json:"skipped,omitempty"`
	// Overwritten lists output-relative files that replaced existing ones
	// (ConflictOverwrite only).
	Overwritten []string `json:"overwritten,omitempty"`
	// Warnings lists non-fatal skips (e.g. symlinks).
	Warnings []string `json:"warnings,omitempty"`
}

// Generate materializes the template under targetParent. It is
// GenerateContext with a background context.
func Generate(t *Template, targetParent string, values, extra map[string]string, opts *Options) (*Result, error) {
	return GenerateContext(context.Background(), t, targetParent, values, extra, opts)
}

// GenerateContext materializes the template under targetParent.
//
// values holds the user's answers keyed by declared parameter name
// (case-insensitive; missing → DefaultValue → ""). extra holds caller context
// parameters (e.g. BRUV's bruvBrand/bruvDate) resolvable in names and content
// without being declared — declared parameters with the same name win.
//
// The generated root is targetParent/<renamed template root name>. Whether it
// may already exist is decided by Options.Conflict (default: refuse).
// targetParent is created if missing. Generating into the template folder
// itself is refused (recursion guard). Cancelling ctx stops between entries;
// whatever was written so far stays on disk.
func GenerateContext(ctx context.Context, t *Template, targetParent string, values, extra map[string]string, opts *Options) (*Result, error) {
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

	rootPath := filepath.Join(targetParent, plan.RootName)
	rootExists := false
	if info, err := os.Stat(rootPath); err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("target %q exists and is not a folder: %w", rootPath, ErrTargetExists)
		}
		if o.Conflict == ConflictRefuse {
			return nil, fmt.Errorf("target %q: %w — refusing to generate into it", rootPath, ErrTargetExists)
		}
		rootExists = true
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(targetParent, 0o755); err != nil {
		return nil, err
	}
	if !rootExists {
		if err := os.Mkdir(rootPath, 0o755); err != nil {
			return nil, err
		}
	}

	res := &Result{RootPath: rootPath, Warnings: plan.Warnings}
	total := len(plan.Entries)
	for i, e := range plan.Entries {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		dst := filepath.Join(rootPath, filepath.FromSlash(e.OutputRel))
		src := filepath.Join(t.dir, filepath.FromSlash(e.SourceRel))
		if e.IsDir {
			if err := mkdirForPolicy(dst, rootExists); err != nil {
				return res, err
			}
		} else if err := writeFileForPolicy(res, e, src, dst, bindings, o, rootExists); err != nil {
			return res, err
		}
		if o.Progress != nil {
			o.Progress(i+1, total, e.OutputRel)
		}
	}
	return res, nil
}

// mkdirForPolicy creates dst. Inside a pre-existing root (merge/overwrite) an
// existing folder is fine; an existing file in its place is not.
func mkdirForPolicy(dst string, rootExists bool) error {
	if !rootExists {
		return os.Mkdir(dst, 0o755)
	}
	if info, err := os.Stat(dst); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%q exists as a file where the template has a folder", dst)
		}
		return nil
	}
	return os.Mkdir(dst, 0o755)
}

func writeFileForPolicy(res *Result, e planEntry, src, dst string, bindings []binding, o Options, rootExists bool) error {
	overwrite := false
	if rootExists {
		if info, err := os.Stat(dst); err == nil {
			if info.IsDir() {
				return fmt.Errorf("%q exists as a folder where the template has a file", dst)
			}
			if o.Conflict == ConflictMerge {
				res.Skipped = append(res.Skipped, e.OutputRel)
				return nil
			}
			overwrite = true
		}
	}
	var err error
	if e.Process {
		err = processContentFile(src, dst, bindings, o.ContentSizeLimit, overwrite)
	} else {
		err = copyFile(src, dst, overwrite)
	}
	if err != nil {
		return err
	}
	res.FilesWritten++
	if overwrite {
		res.Overwritten = append(res.Overwritten, e.OutputRel)
	}
	return nil
}
