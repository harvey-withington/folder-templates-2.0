package foldertemplate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// planEntry is one output item, computed before anything is written. Building
// the full plan first gives Preview for free and lets collisions fail the run
// before a single byte lands on disk.
type planEntry struct {
	SourceRel string // path relative to the template dir ("" = root)
	OutputRel string // path relative to the generated root, after renaming
	IsDir     bool
	Process   bool // content token replacement + .ft$ strip applied
}

type generationPlan struct {
	RootName string // renamed template root folder name
	Entries  []planEntry
	Warnings []string
}

// buildPlan walks the template folder, applies name replacements per path
// segment, strips .ft$ from processed files, skips .ft/ config dirs, skips
// symlinks (with a warning — never followed), and rejects case-only output
// collisions (they silently merge on case-insensitive filesystems).
func buildPlan(t *Template, bindings []binding) (*generationPlan, error) {
	rootName, err := applyToName(filepath.Base(t.dir), bindings)
	if err != nil {
		return nil, err
	}
	if rootName == "" || rootName == "." {
		return nil, fmt.Errorf("template root folder name %q resolves to an empty name", filepath.Base(t.dir))
	}

	plan := &generationPlan{RootName: rootName}
	// seen maps lowercased output paths → original-case output path, to flag
	// case-only collisions after renaming (consistent with spec §13).
	seen := map[string]string{}

	walkErr := filepath.WalkDir(t.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(t.dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil // root handled via RootName
		}
		if d.IsDir() && d.Name() == ConfigDirName {
			return filepath.SkipDir // original skip rule ^\.ft$ — never copied
		}
		if d.Type()&fs.ModeSymlink != 0 {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("skipped symlink %q (symlinks are never followed)", rel))
			return nil
		}

		// Rename every segment of the output path; the parent segments were
		// already renamed when their entries were visited, so renaming just
		// this entry's own name and joining with the renamed parent is enough.
		outParent := ""
		if parentRel := filepath.Dir(rel); parentRel != "." {
			p, ok := seen[strings.ToLower(filepath.ToSlash(parentRel))+"\x00src"]
			if !ok {
				return fmt.Errorf("internal: parent of %q not planned", rel)
			}
			outParent = p
		}

		process := false
		name := d.Name()
		if !d.IsDir() {
			renamed, err := applyToName(name, bindings)
			if err != nil {
				return err
			}
			// Spec: "the .ft$ extension is stripped from the output filename"
			// — the check and strip apply to the renamed output name.
			if strings.HasSuffix(renamed, ContentExt) {
				renamed = strings.TrimSuffix(renamed, ContentExt)
				process = true
			}
			name = renamed
		} else {
			renamed, err := applyToName(name, bindings)
			if err != nil {
				return err
			}
			name = renamed
		}
		if name == "" {
			return fmt.Errorf("output name for %q resolves to an empty name", rel)
		}

		outRel := name
		if outParent != "" {
			outRel = outParent + "/" + name
		}

		lower := strings.ToLower(outRel)
		if prev, dup := seen[lower]; dup {
			return fmt.Errorf("case-only collision in output: %q and %q map to the same path on case-insensitive filesystems", prev, outRel)
		}
		seen[lower] = outRel
		// Index dirs by their source path too, so children can find their
		// renamed parent (the "\x00src" suffix cannot collide with real paths).
		if d.IsDir() {
			seen[strings.ToLower(filepath.ToSlash(rel))+"\x00src"] = outRel
		}

		plan.Entries = append(plan.Entries, planEntry{
			SourceRel: filepath.ToSlash(rel),
			OutputRel: outRel,
			IsDir:     d.IsDir(),
			Process:   process,
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return plan, nil
}

// guardTarget rejects generating into the template itself (open infinite-
// recursion bug in the C# original). Comparison is case-insensitive — a false
// positive on a pathological case-sensitive path is safer than recursion.
func guardTarget(templateDir, targetParent string) error {
	src, err := filepath.Abs(templateDir)
	if err != nil {
		return err
	}
	dst, err := filepath.Abs(targetParent)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(strings.ToLower(src), strings.ToLower(dst))
	if err != nil {
		return nil // different volumes — cannot be nested
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return fmt.Errorf("target %q is inside the template folder %q — refusing to generate (would recurse)", targetParent, templateDir)
	}
	return nil
}
