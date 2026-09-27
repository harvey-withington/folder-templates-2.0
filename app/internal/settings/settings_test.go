package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestOpenMissingGivesDefaults(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "sub", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := st.Get()
	if s.Theme != "system" || s.DefaultConflict != "refuse" || !s.OpenFolderAfter || s.LibraryFolders == nil {
		t.Errorf("defaults = %+v", s)
	}
}

func TestUpdatePersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "settings.json")
	st, _ := Open(path)
	_, err := st.Update(func(s *Settings) {
		s.Theme = "light"
		s.LibraryFolders = []string{`C:\T`, `c:\t\`, " ", `D:\U`}
		s.TouchRecent(`C:\T\x`, time.Unix(100, 0))
		s.Window = &WindowBounds{X: 5, Y: 6, Width: 900, Height: 600}
	})
	if err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s := again.Get()
	wantFolders := 3
	if runtime.GOOS == "windows" {
		wantFolders = 2 // C:\T and c:\t\ are the same folder
	}
	if s.Theme != "light" || len(s.LibraryFolders) != wantFolders || len(s.Recents) != 1 || s.Window == nil || s.Window.Width != 900 {
		t.Errorf("reloaded = %+v", s)
	}
}

func TestSanitizeRejectsJunk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"neon","defaultConflict":"yolo","window":{"width":10,"height":10}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, _ := Open(path)
	s := st.Get()
	if s.Theme != "system" || s.DefaultConflict != "refuse" || s.Window != nil || !s.ShowSamples {
		t.Errorf("sanitized = %+v", s)
	}
}

func TestCorruptFileIsSetAside(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Open(path)
	if err != nil || st.Get().Theme != "system" {
		t.Fatalf("open corrupt: %v", err)
	}
	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Error("corrupt file should be kept as settings.json.bad")
	}
}

func TestRecentsPinsAndTargets(t *testing.T) {
	var s Settings = Defaults()
	for i := 0; i < maxRecents+5; i++ {
		s.TouchRecent(filepath.Join("r", string(rune('a'+i%26)), string(rune('a'+i/26))), time.Unix(int64(i), 0))
	}
	s = sanitize(s)
	if len(s.Recents) != maxRecents {
		t.Errorf("recents capped at %d, got %d", maxRecents, len(s.Recents))
	}
	s.TouchRecent("x", time.Unix(1, 0))
	s.TouchRecent("y", time.Unix(2, 0))
	s.TouchRecent("x", time.Unix(3, 0))
	if s.Recents[0].Path != "x" || s.Recents[1].Path != "y" {
		t.Errorf("most recent first: %+v", s.Recents[:2])
	}
	s.RemoveRecent("y")
	if s.Recents[1].Path == "y" {
		t.Error("RemoveRecent")
	}
	s.SetPinned("p", true)
	s.SetPinned("p", true)
	s.SetPinned("q", true)
	s.SetPinned("p", false)
	if len(s.Pinned) != 1 || s.Pinned[0] != "q" {
		t.Errorf("pins = %v", s.Pinned)
	}
	s.TouchTarget("t1")
	s.TouchTarget("t2")
	s.TouchTarget("t1")
	if s.RecentTargets[0] != "t1" || len(s.RecentTargets) != 2 {
		t.Errorf("targets = %v", s.RecentTargets)
	}
}

func TestReplaceKeepsRecentsAndWindow(t *testing.T) {
	st, _ := Open(filepath.Join(t.TempDir(), "s.json"))
	st.Update(func(s *Settings) {
		s.TouchRecent("keep", time.Unix(1, 0))
		s.Window = &WindowBounds{Width: 800, Height: 600}
	})
	page := Defaults()
	page.Theme = "dark"
	got, err := st.Replace(page)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "dark" || len(got.Recents) != 1 || got.Window == nil {
		t.Errorf("replace = %+v", got)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	st, _ := Open(filepath.Join(t.TempDir(), "s.json"))
	s := st.Get()
	s.LibraryFolders = append(s.LibraryFolders, "mutated")
	if len(st.Get().LibraryFolders) != 0 {
		t.Error("Get must not expose internal slices")
	}
}
