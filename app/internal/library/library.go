// Package library finds the templates shown on the home screen: every
// template inside the configured library folders, plus pinned and recently
// used templates wherever they live, plus the bundled samples.
package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ft "github.com/harvey-withington/foldertemplate"

	"github.com/harvey-withington/folder-templates/app/internal/settings"
)

// Source says why a template is listed.
type Source string

const (
	SourceLibrary Source = "library"
	SourceRecent  Source = "recent"
	SourcePinned  Source = "pinned"
	SourceSample  Source = "sample"
)

// Entry is one template on the home screen.
type Entry struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FolderName  string    `json:"folderName"`
	ParamCount  int       `json:"paramCount"`
	Source      Source    `json:"source"`
	Pinned      bool      `json:"pinned"`
	LastUsed    time.Time `json:"lastUsed,omitzero"`
	// LibraryFolder is the configured folder it was found in, if any.
	LibraryFolder string `json:"libraryFolder,omitempty"`
	// Missing: a pinned or recent template whose folder is gone.
	Missing bool `json:"missing,omitempty"`
	// Error: the descriptor exists but can't be read.
	Error string `json:"error,omitempty"`
}

// Input is what List works from.
type Input struct {
	LibraryFolders []string
	Pinned         []string
	Recents        []settings.Recent
	// SamplesDir, when set, is scanned like a library folder.
	SamplesDir string
}

// MaxDepth bounds how deep List looks for templates under a library folder.
// Templates are not searched inside other templates.
const MaxDepth = 4

// List gathers entries: pinned first, then recents (most recent first), then
// library and sample templates by name. Each template appears once.
func List(in Input) []Entry {
	var out []Entry
	seen := map[string]int{} // normalized path → index in out
	key := func(p string) string {
		k := filepath.Clean(p)
		if filepath.Separator == '\\' {
			k = strings.ToLower(k)
		}
		return k
	}
	add := func(e Entry) {
		k := key(e.Path)
		if i, ok := seen[k]; ok {
			// Keep the first (highest-priority) listing, but merge facts.
			if out[i].LastUsed.IsZero() {
				out[i].LastUsed = e.LastUsed
			}
			if out[i].LibraryFolder == "" {
				out[i].LibraryFolder = e.LibraryFolder
			}
			return
		}
		seen[k] = len(out)
		out = append(out, e)
	}

	pinned := map[string]bool{}
	for _, p := range in.Pinned {
		pinned[key(p)] = true
	}
	lastUsed := map[string]time.Time{}
	for _, r := range in.Recents {
		lastUsed[key(r.Path)] = r.LastUsed
	}

	for _, p := range in.Pinned {
		e := describe(p, SourcePinned)
		e.Pinned = true
		e.LastUsed = lastUsed[key(p)]
		add(e)
	}
	for _, r := range in.Recents {
		e := describe(r.Path, SourceRecent)
		e.LastUsed = r.LastUsed
		e.Pinned = pinned[key(r.Path)]
		add(e)
	}

	var found []Entry
	scan := func(root string, src Source) {
		for _, dir := range findTemplates(root) {
			e := describe(dir, src)
			e.LibraryFolder = root
			e.Pinned = pinned[key(dir)]
			e.LastUsed = lastUsed[key(dir)]
			found = append(found, e)
		}
	}
	for _, root := range in.LibraryFolders {
		scan(root, SourceLibrary)
	}
	if in.SamplesDir != "" {
		scan(in.SamplesDir, SourceSample)
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Source != found[j].Source {
			return found[i].Source == SourceLibrary
		}
		return strings.ToLower(found[i].Name) < strings.ToLower(found[j].Name)
	})
	for _, e := range found {
		add(e)
	}
	if out == nil {
		out = []Entry{}
	}
	return out
}

func describe(dir string, src Source) Entry {
	e := Entry{Path: dir, FolderName: filepath.Base(dir), Name: filepath.Base(dir), Source: src}
	tpl, err := ft.Load(dir)
	switch {
	case err == nil:
		if tpl.Name != "" {
			e.Name = tpl.Name
		}
		e.Description = tpl.Description
		e.ParamCount = len(tpl.Parameters)
	case errors.Is(err, ft.ErrNotATemplate):
		if _, statErr := os.Stat(dir); statErr != nil {
			e.Missing = true
		} else {
			e.Error = err.Error()
		}
	default:
		e.Error = err.Error()
	}
	return e
}

// findTemplates returns template folders under root (root itself included),
// not descending into a template once found, nor into hidden folders.
func findTemplates(root string) []string {
	var out []string
	root = filepath.Clean(root)
	baseDepth := strings.Count(root, string(filepath.Separator))
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if isTemplate(path) {
			out = append(out, path)
			return fs.SkipDir
		}
		if strings.Count(path, string(filepath.Separator))-baseDepth >= MaxDepth {
			return fs.SkipDir
		}
		return nil
	})
	return out
}

func isTemplate(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ft.ConfigDirName, ft.ConfigFileName))
	return err == nil && !info.IsDir()
}
