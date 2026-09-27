package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestParityWith1_0 runs each testdata/parity case through the 1.0-compatible
// command line and compares the output tree with what Folder Templates 1.0
// produces.
func TestParityWith1_0(t *testing.T) {
	cases, err := filepath.Glob(filepath.Join("..", "..", "testdata", "parity", "*", "case.json"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no parity cases found: %v", err)
	}
	for _, caseFile := range cases {
		caseDir := filepath.Dir(caseFile)
		t.Run(filepath.Base(caseDir), func(t *testing.T) {
			var spec struct {
				Values    map[string]string `json:"values"`
				EmptyDirs []string          `json:"emptyDirs"`
			}
			if err := json.Unmarshal([]byte(readString(t, caseFile)), &spec); err != nil {
				t.Fatal(err)
			}
			roots, _ := os.ReadDir(filepath.Join(caseDir, "template"))
			if len(roots) != 1 {
				t.Fatalf("template/ must hold exactly one folder, has %d", len(roots))
			}
			src := copyTree(t, filepath.Join(caseDir, "template", roots[0].Name()))
			for _, d := range spec.EmptyDirs {
				if err := os.MkdirAll(filepath.Join(src, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			target := t.TempDir()
			args := []string{"-sourceFolder", src, "-targetFolder", target, "-noprompt", "-nowait"}
			names := make([]string, 0, len(spec.Values))
			for k := range spec.Values {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				args = append(args, "-"+k, spec.Values[k])
			}
			if r := runFT(t, "", false, args...); r.code != 0 {
				t.Fatalf("ft exited %d:\n%s%s", r.code, r.stdout, r.stderr)
			}

			var wantManifest []string
			for _, line := range strings.Split(readString(t, filepath.Join(caseDir, "manifest.txt")), "\n") {
				if line = strings.TrimRight(line, "\r"); line != "" {
					wantManifest = append(wantManifest, line)
				}
			}
			sort.Strings(wantManifest)
			gotManifest := manifest(t, target)
			if strings.Join(gotManifest, "\n") != strings.Join(wantManifest, "\n") {
				t.Fatalf("output tree differs\n got: %q\nwant: %q", gotManifest, wantManifest)
			}

			expected := filepath.Join(caseDir, "expected")
			for _, rel := range gotManifest {
				if strings.HasSuffix(rel, "/") {
					continue
				}
				want, err := os.ReadFile(filepath.Join(expected, filepath.FromSlash(rel)))
				if err != nil {
					t.Errorf("%s: no expected content: %v", rel, err)
					continue
				}
				got, _ := os.ReadFile(filepath.Join(target, filepath.FromSlash(rel)))
				if string(got) != string(want) {
					t.Errorf("%s differs\n got: %q\nwant: %q", rel, got, want)
				}
			}
		})
	}
}

func readString(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// manifest lists every path under root, slash-separated, folders ending in /.
func manifest(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// copyTree copies a fixture template into a temp dir so tests never write
// next to the committed fixture.
func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}
