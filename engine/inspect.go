package foldertemplate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Inspection describes a folder that may or may not be a template yet — the
// shape a UI needs to decide between "generate from it" and "make it one".
type Inspection struct {
	Dir        string `json:"dir"`
	FolderName string `json:"folderName"`
	IsTemplate bool   `json:"isTemplate"`
	// Template is the loaded descriptor when IsTemplate.
	Template *Template `json:"template,omitempty"`
	// LoadError is set when .ft/template.json exists but cannot be parsed.
	LoadError string `json:"loadError,omitempty"`
	// Issues are Validate's findings, as text.
	Issues []string `json:"issues"`
	// DefaultTarget is ResolveDefaultTarget for templates.
	DefaultTarget string `json:"defaultTarget,omitempty"`
	Files         int    `json:"files"`
	Dirs          int    `json:"dirs"`
	SizeBytes     int64  `json:"sizeBytes"`
}

// Inspect examines dir: whether it holds a template, whether the descriptor
// validates, and how big the folder is (excluding .ft/). It fails only when
// dir itself is unusable; a broken descriptor is reported in LoadError.
func Inspect(dir string) (*Inspection, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", dir)
	}
	insp := &Inspection{Dir: abs, FolderName: filepath.Base(abs), Issues: []string{}}

	tpl, err := Load(abs)
	switch {
	case err == nil:
		insp.IsTemplate = true
		if tpl.Parameters == nil {
			tpl.Parameters = []Parameter{}
		}
		insp.Template = tpl
		insp.DefaultTarget = ResolveDefaultTarget(tpl)
		for _, issue := range tpl.Validate() {
			insp.Issues = append(insp.Issues, issue.Error())
		}
	case errors.Is(err, ErrNotATemplate):
	default:
		insp.LoadError = err.Error()
	}

	entries, err := Tree(abs)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir {
			insp.Dirs++
		} else {
			insp.Files++
			insp.SizeBytes += e.Size
		}
	}
	return insp, nil
}

// TreeEntry is one item of a template folder's source tree.
type TreeEntry struct {
	Rel       string `json:"rel"` // slash-separated, relative to the template folder
	IsDir     bool   `json:"isDir"`
	Size      int64  `json:"size"`
	Processed bool   `json:"processed"` // name ends in .ft$ — content replacement on
}

// Tree lists dir's source tree in lexical walk order, excluding .ft/ and
// symlinks — exactly the items a generation would consider.
func Tree(dir string) ([]TreeEntry, error) {
	out := []TreeEntry{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if d.IsDir() && d.Name() == ConfigDirName {
			return filepath.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		e := TreeEntry{Rel: filepath.ToSlash(rel), IsDir: d.IsDir()}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			e.Size = info.Size()
			e.Processed = strings.HasSuffix(d.Name(), ContentExt)
		}
		out = append(out, e)
		return nil
	})
	return out, err
}

// SetContentProcessing turns content replacement on or off for one file by
// renaming it to add or strip the .ft$ suffix. rel is relative to the template
// folder. It returns the file's new relative path (unchanged when the file is
// already in the requested state) and refuses to overwrite an existing file.
func SetContentProcessing(dir, rel string, on bool) (string, error) {
	src, cleanRel, err := resolveInside(dir, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", rel)
	}
	has := strings.HasSuffix(cleanRel, ContentExt)
	if has == on {
		return cleanRel, nil
	}
	newRel := cleanRel + ContentExt
	if !on {
		newRel = strings.TrimSuffix(cleanRel, ContentExt)
		if newRel == "" || strings.HasSuffix(newRel, "/") {
			return "", fmt.Errorf("%s: stripping %s would leave an empty name", rel, ContentExt)
		}
	}
	dst := filepath.Join(filepath.Dir(src), filepath.Base(filepath.FromSlash(newRel)))
	if _, err := os.Lstat(dst); err == nil {
		return "", fmt.Errorf("cannot rename %s: %s already exists", cleanRel, newRel)
	}
	if err := os.Rename(src, dst); err != nil {
		return "", err
	}
	return newRel, nil
}

// resolveInside joins a slash-separated rel onto dir, rejecting absolute
// paths, escapes via .., and anything inside .ft/.
func resolveInside(dir, rel string) (abs, cleanRel string, err error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	c := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if c == "." || c == "" || filepath.IsAbs(filepath.FromSlash(rel)) || c == ".." || strings.HasPrefix(c, "../") {
		return "", "", fmt.Errorf("%q is not a path inside the template", rel)
	}
	if c == ConfigDirName || strings.HasPrefix(c, ConfigDirName+"/") {
		return "", "", fmt.Errorf("%q is inside the template's %s folder", rel, ConfigDirName)
	}
	return filepath.Join(root, filepath.FromSlash(c)), c, nil
}
