//go:build windows

package shell

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

// testIntegration points at a temp SendTo folder and a scratch HKCU key that
// is deleted afterwards, so tests never touch the real Explorer setup.
func testIntegration(t *testing.T) *Integration {
	t.Helper()
	root := `Software\FolderTemplatesTest\` + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() {
		_ = deleteTree(registry.CURRENT_USER, root)
		// Drop the shared parent too once no other test run is using it.
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\FolderTemplatesTest`)
	})
	return &Integration{
		Exe:        filepath.Join(t.TempDir(), "FolderTemplates.exe"),
		SendToDir:  t.TempDir(),
		ClassesKey: root + `\Classes`,
	}
}

func TestShellFeaturesToggle(t *testing.T) {
	in := testIntegration(t)
	for _, f := range Features {
		if in.Status()[f] {
			t.Fatalf("%s on before setting", f)
		}
		if err := in.Set(f, true); err != nil {
			t.Fatalf("enable %s: %v", f, err)
		}
		if !in.Status()[f] {
			t.Errorf("%s off after enabling", f)
		}
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, in.ClassesKey+`\Directory\Background\shell\`+verbNewHere+`\command`, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	cmd, _, _ := k.GetStringValue("")
	k.Close()
	if want := `"` + in.Exe + `" --pick-template --target "%V"`; cmd != want {
		t.Errorf("background command = %q, want %q", cmd, want)
	}

	if err := in.RemoveAll(); err != nil {
		t.Fatal(err)
	}
	for f, on := range in.Status() {
		if on {
			t.Errorf("%s still on after RemoveAll", f)
		}
	}
	if err := in.RemoveAll(); err != nil {
		t.Errorf("removing twice must be fine: %v", err)
	}
}

func TestForeignShortcutCountsAsOff(t *testing.T) {
	in := testIntegration(t)
	if err := in.Set(SendToProcess, true); err != nil {
		t.Fatal(err)
	}
	other := *in
	other.Exe = filepath.Join(t.TempDir(), "FolderTemplates.App.exe") // a 1.0 install
	if other.Status()[SendToProcess] {
		t.Error("a shortcut pointing at another exe must count as off")
	}
	if err := other.Set(SendToProcess, true); err != nil {
		t.Fatal(err)
	}
	if !other.Status()[SendToProcess] || in.Status()[SendToProcess] {
		t.Error("enabling must take over the same-named shortcut")
	}
	entries, _ := os.ReadDir(in.SendToDir)
	if len(entries) != 1 {
		t.Errorf("want exactly one shortcut, got %d", len(entries))
	}
}
