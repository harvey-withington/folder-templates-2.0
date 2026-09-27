package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	ft "github.com/harvey-withington/folder-templates-2.0/engine"

	"github.com/harvey-withington/folder-templates-2.0/app/internal/settings"
)

func mkTemplate(t *testing.T, dir, name string) string {
	t.Helper()
	if err := ft.Save(&ft.Template{Name: name, Description: name + " desc", Parameters: []ft.Parameter{{Name: "a"}}}, dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestListFindsAndOrders(t *testing.T) {
	lib := t.TempDir()
	zeta := mkTemplate(t, filepath.Join(lib, "z"), "Zeta")
	alpha := mkTemplate(t, filepath.Join(lib, "group", "a"), "alpha")
	// a template nested inside another template is part of it, not listed
	mkTemplate(t, filepath.Join(zeta, "inner"), "Inner")
	// hidden folders are skipped
	mkTemplate(t, filepath.Join(lib, ".hidden", "h"), "Hidden")
	// too deep
	mkTemplate(t, filepath.Join(lib, "1", "2", "3", "4", "5"), "Deep")

	samples := t.TempDir()
	sample := mkTemplate(t, filepath.Join(samples, "s"), "Sample")

	elsewhere := mkTemplate(t, filepath.Join(t.TempDir(), "e"), "Elsewhere")
	gone := filepath.Join(t.TempDir(), "gone")

	now := time.Unix(1000, 0)
	got := List(Input{
		LibraryFolders: []string{lib},
		Pinned:         []string{alpha},
		Recents:        []settings.Recent{{Path: elsewhere, LastUsed: now}, {Path: gone, LastUsed: now}, {Path: zeta, LastUsed: now}},
		SamplesDir:     samples,
	})

	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	want := []string{"alpha", "Elsewhere", "gone", "Zeta", "Sample"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
	if !got[0].Pinned || got[0].Source != SourcePinned || got[0].LibraryFolder != lib || got[0].ParamCount != 1 {
		t.Errorf("pinned entry = %+v", got[0])
	}
	if !got[2].Missing {
		t.Errorf("gone recent should be Missing: %+v", got[2])
	}
	if got[3].Source != SourceRecent || got[3].LibraryFolder != lib {
		t.Errorf("recent that is also in the library keeps recent priority and learns its folder: %+v", got[3])
	}
	if got[4].Source != SourceSample || got[4].Path != sample {
		t.Errorf("sample = %+v", got[4])
	}
}

func TestListReportsBrokenDescriptor(t *testing.T) {
	lib := t.TempDir()
	bad := filepath.Join(lib, "bad")
	if err := os.MkdirAll(filepath.Join(bad, ".ft"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, ".ft", "template.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := List(Input{LibraryFolders: []string{lib}})
	if len(got) != 1 || got[0].Error == "" {
		t.Errorf("broken = %+v", got)
	}
}

func TestListEmpty(t *testing.T) {
	got := List(Input{LibraryFolders: []string{filepath.Join(t.TempDir(), "nope")}})
	if got == nil || len(got) != 0 {
		t.Errorf("want empty non-nil list, got %#v", got)
	}
}
