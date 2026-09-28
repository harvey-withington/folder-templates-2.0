// Package settings persists the app's preferences, recents and window
// bounds as JSON — in %AppData%\FolderTemplates, or next to the executable in
// portable mode (a file named "portable" beside it).
package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	fileName       = "settings.json"
	appDirName     = "FolderTemplates"
	portableMarker = "portable"

	maxRecents       = 30
	maxRecentTargets = 10
)

// Recent is a template the user opened or generated from.
type Recent struct {
	Path     string    `json:"path"`
	LastUsed time.Time `json:"lastUsed"`
}

// WindowBounds is the last window size, in device-independent pixels. The
// position is deliberately not kept: the window opens on the display it was
// launched from (see internal/placement).
type WindowBounds struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

// Settings is everything persisted.
type Settings struct {
	// Theme is "system", "dark" or "light".
	Theme          string   `json:"theme"`
	LibraryFolders []string `json:"libraryFolders"`
	Recents        []Recent `json:"recents"`
	Pinned         []string `json:"pinned"`
	RecentTargets  []string `json:"recentTargets"`
	// DefaultConflict is the conflict policy the generate view starts with.
	DefaultConflict string `json:"defaultConflict"`
	// OpenFolderAfter opens the generated folder in Explorer when done.
	OpenFolderAfter bool `json:"openFolderAfter"`
	// CloseAfter closes the window when done (1.0's "exit immediately").
	CloseAfter bool `json:"closeAfter"`
	// ShowSamples lists the bundled sample templates in the library.
	ShowSamples bool          `json:"showSamples"`
	Window      *WindowBounds `json:"window,omitempty"`
}

// Defaults are used for a first run and fill fields missing from older files.
func Defaults() Settings {
	return Settings{
		Theme:           "system",
		LibraryFolders:  []string{},
		Recents:         []Recent{},
		Pinned:          []string{},
		RecentTargets:   []string{},
		DefaultConflict: "refuse",
		OpenFolderAfter: true,
		ShowSamples:     true,
	}
}

// Store loads and saves Settings at one path, safe for concurrent use.
type Store struct {
	mu   sync.Mutex
	path string
	s    Settings
}

// DefaultPath picks the settings file for the running executable: beside it
// in portable mode, otherwise in the user's config directory.
func DefaultPath() (string, error) {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, portableMarker)); err == nil {
			return filepath.Join(dir, fileName), nil
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, appDirName, fileName), nil
}

// Open loads path, starting from Defaults when it doesn't exist yet. A file
// that can't be parsed is set aside as settings.json.bad rather than lost.
func Open(path string) (*Store, error) {
	st := &Store{path: path, s: Defaults()}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	// Editors like Notepad and Windows PowerShell save UTF-8 with a BOM,
	// which encoding/json rejects; a hand-edited file must still load.
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	loaded := Defaults()
	if err := json.Unmarshal(raw, &loaded); err != nil {
		_ = os.Rename(path, path+".bad")
		return st, nil
	}
	st.s = sanitize(loaded)
	return st, nil
}

// Path is the file the store reads and writes.
func (st *Store) Path() string { return st.path }

// Get returns a copy of the current settings.
func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return clone(st.s)
}

// Update applies fn to a copy, then saves the result.
func (st *Store) Update(fn func(*Settings)) (Settings, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	next := clone(st.s)
	fn(&next)
	next = sanitize(next)
	if err := st.write(next); err != nil {
		return clone(st.s), err
	}
	st.s = next
	return clone(next), nil
}

// Replace swaps in s wholesale (the settings page's save), keeping fields the
// page doesn't manage — recents, pins and window bounds — from the store.
func (st *Store) Replace(s Settings) (Settings, error) {
	return st.Update(func(cur *Settings) {
		keep := *cur
		*cur = s
		cur.Recents, cur.Pinned, cur.RecentTargets, cur.Window = keep.Recents, keep.Pinned, keep.RecentTargets, keep.Window
	})
}

func (st *Store) write(s Settings) error {
	if err := os.MkdirAll(filepath.Dir(st.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}

// TouchRecent moves path to the front of the recents.
func (s *Settings) TouchRecent(path string, now time.Time) {
	out := []Recent{{Path: path, LastUsed: now}}
	for _, r := range s.Recents {
		if !SamePath(r.Path, path) {
			out = append(out, r)
		}
	}
	s.Recents = out
}

// RemoveRecent drops path from the recents.
func (s *Settings) RemoveRecent(path string) {
	out := s.Recents[:0:0]
	for _, r := range s.Recents {
		if !SamePath(r.Path, path) {
			out = append(out, r)
		}
	}
	s.Recents = out
}

// SetPinned pins or unpins path.
func (s *Settings) SetPinned(path string, pinned bool) {
	out := []string{}
	for _, p := range s.Pinned {
		if !SamePath(p, path) {
			out = append(out, p)
		}
	}
	if pinned {
		out = append(out, path)
	}
	s.Pinned = out
}

// TouchTarget moves dir to the front of the recent destinations.
func (s *Settings) TouchTarget(dir string) {
	out := []string{dir}
	for _, t := range s.RecentTargets {
		if !SamePath(t, dir) {
			out = append(out, t)
		}
	}
	s.RecentTargets = out
}

// SamePath compares paths the way the OS does: case-insensitively on
// Windows, and ignoring trailing separators.
func SamePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func sanitize(s Settings) Settings {
	switch s.Theme {
	case "system", "dark", "light":
	default:
		s.Theme = "system"
	}
	switch s.DefaultConflict {
	case "refuse", "merge", "overwrite":
	default:
		s.DefaultConflict = "refuse"
	}
	s.LibraryFolders = dedupe(s.LibraryFolders)
	s.Pinned = dedupe(s.Pinned)
	s.RecentTargets = dedupe(s.RecentTargets)
	if len(s.RecentTargets) > maxRecentTargets {
		s.RecentTargets = s.RecentTargets[:maxRecentTargets]
	}
	if s.Recents == nil {
		s.Recents = []Recent{}
	}
	if len(s.Recents) > maxRecents {
		s.Recents = s.Recents[:maxRecents]
	}
	if s.Window != nil && (s.Window.Width < 480 || s.Window.Height < 360) {
		s.Window = nil
	}
	return s
}

func dedupe(list []string) []string {
	out := []string{}
	for _, p := range list {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		dup := false
		for _, q := range out {
			if SamePath(p, q) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, p)
		}
	}
	return out
}

func clone(s Settings) Settings {
	c := s
	c.LibraryFolders = append([]string{}, s.LibraryFolders...)
	c.Recents = append([]Recent{}, s.Recents...)
	c.Pinned = append([]string{}, s.Pinned...)
	c.RecentTargets = append([]string{}, s.RecentTargets...)
	if s.Window != nil {
		w := *s.Window
		c.Window = &w
	}
	return c
}
