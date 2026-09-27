package foldertemplate_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	ft "github.com/harvey-withington/foldertemplate"
)

// --- helpers -----------------------------------------------------------------

func str(s string) *string { return &s }

// makeTemplate builds a template folder under a temp dir. rootName is the
// template folder's own name (renaming applies to it too). files maps
// relative paths to content; keys ending in "/" create directories.
func makeTemplate(t *testing.T, rootName string, tpl ft.Template, files map[string][]byte) *ft.Template {
	t.Helper()
	dir := filepath.Join(t.TempDir(), rootName)
	if err := ft.Save(&tpl, dir); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if strings.HasSuffix(rel, "/") {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := ft.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func readOut(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

// --- fixture (authored C#-style: PascalCase JSON) ------------------------------

func TestLoadFixturePascalCase(t *testing.T) {
	tpl, err := ft.Load(filepath.Join("testdata", "youtube-template"))
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "YouTube Video Project" {
		t.Errorf("Name = %q", tpl.Name)
	}
	if len(tpl.Parameters) != 2 {
		t.Fatalf("Parameters = %d, want 2", len(tpl.Parameters))
	}
	p := tpl.Parameters[0]
	if p.Name != "videoName" || p.Internal() || !p.ReplaceInFileNames || !p.ReplaceInFiles {
		t.Errorf("videoName parsed wrong: %+v", p)
	}
	if p.Match == nil || *p.Match != `\{videoName\}` {
		t.Errorf("Match = %v", p.Match)
	}
	ch := tpl.Parameters[1]
	if !ch.Internal() || ch.DefaultValue == nil || *ch.DefaultValue != "OOP" {
		t.Errorf("channel should be internal with default OOP: %+v", ch)
	}
	if issues := tpl.Validate(); len(issues) != 0 {
		t.Errorf("Validate: %v", issues)
	}
}

func TestGenerateFixture(t *testing.T) {
	tpl, err := ft.Load(filepath.Join("testdata", "youtube-template"))
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	res, err := ft.Generate(tpl, target,
		map[string]string{"videoName": "episode-042"},
		map[string]string{"bruvDate": "2026-07-04"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(res.RootPath) != "youtube-template" {
		t.Errorf("root = %q", res.RootPath)
	}

	if _, err := os.Stat(filepath.Join(res.RootPath, ".ft")); !os.IsNotExist(err) {
		t.Error(".ft/ must never be copied to output")
	}
	if _, err := os.Stat(filepath.Join(res.RootPath, "episode-042 assets", "b-roll", "placeholder.txt")); err != nil {
		t.Errorf("renamed dir tree missing: %v", err)
	}

	verbatim := readOut(t, res.RootPath, "notes episode-042.md")
	if !strings.Contains(verbatim, "{{$videoName}}") {
		t.Error("token in non-.ft$ file must pass through verbatim")
	}

	script := readOut(t, res.RootPath, "script.md")
	for _, want := range []string{
		"# episode-042",
		"Channel: OOP",     // internal param default
		"Date: 2026-07-04", // extra (caller context) param
		"Case-insensitive: episode-042",
		"Unknown token passes through: {{$unknown}}",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script.md missing %q; got:\n%s", want, script)
		}
	}
	if _, err := os.Stat(filepath.Join(res.RootPath, "script.md.ft$")); !os.IsNotExist(err) {
		t.Error(".ft$ suffix must be stripped from output")
	}
}

// --- core semantics ------------------------------------------------------------

func baseParam(name string) ft.Parameter {
	return ft.Parameter{Name: name, Type: "text", Prompt: str(name + "?"), ReplaceInFileNames: true, ReplaceInFiles: true}
}

func TestRootFolderRenamed(t *testing.T) {
	tpl := makeTemplate(t, "{p} project",
		ft.Template{Name: "root", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"a.txt": []byte("x")})
	res, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "ep1"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(res.RootPath) != "ep1 project" {
		t.Errorf("root = %q, want %q", filepath.Base(res.RootPath), "ep1 project")
	}
}

func TestSaveWritesCamelCase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tpl")
	err := ft.Save(&ft.Template{Name: "T", Parameters: []ft.Parameter{baseParam("x")}}, dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".ft", "template.json"))
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name", "description", "defaultTargetPath", "parameters"} {
		if _, ok := keys[want]; !ok {
			t.Errorf("missing camelCase key %q in output", want)
		}
	}
	if _, ok := keys["Name"]; ok {
		t.Error("PascalCase key written — must be camelCase like the C# app's serializer")
	}
	if !strings.Contains(string(raw), `"replaceInFileNames"`) {
		t.Error("parameter keys must be camelCase")
	}
	if _, err := ft.Load(dir); err != nil {
		t.Errorf("round-trip Load failed: %v", err)
	}
}

func TestBinaryCopiedByteForByte(t *testing.T) {
	blob := make([]byte, 4096)
	for i := range blob {
		blob[i] = byte(i % 256)
	}
	tpl := makeTemplate(t, "bin",
		ft.Template{Name: "bin", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"data.bin": blob})
	res, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "v"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(res.RootPath, "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, blob) {
		t.Error("binary file not copied byte-for-byte")
	}
}

func TestMissingValueResolvesEmpty(t *testing.T) {
	tpl := makeTemplate(t, "m",
		ft.Template{Name: "m", Parameters: []ft.Parameter{baseParam("gone")}},
		map[string][]byte{"a-{gone}.md.ft$": []byte("v={{$gone}}!")})
	res, err := ft.Generate(tpl, t.TempDir(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := readOut(t, res.RootPath, "a-.md"); got != "v=!" {
		t.Errorf("missing value should resolve to \"\"; got %q", got)
	}
}

func TestDeclaredParameterBeatsExtra(t *testing.T) {
	p := baseParam("bruvDate")
	p.Prompt = nil // internal
	p.DefaultValue = str("DECLARED")
	tpl := makeTemplate(t, "d",
		ft.Template{Name: "d", Parameters: []ft.Parameter{p}},
		map[string][]byte{"f.md.ft$": []byte("{{$bruvDate}}")})
	res, err := ft.Generate(tpl, t.TempDir(), nil, map[string]string{"bruvDate": "EXTRA"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := readOut(t, res.RootPath, "f.md"); got != "DECLARED" {
		t.Errorf("declared parameter must win over extra; got %q", got)
	}
}

// --- port fixes (spec §6.2) -----------------------------------------------------

func TestRecursionGuard(t *testing.T) {
	tpl := makeTemplate(t, "r",
		ft.Template{Name: "r"},
		map[string][]byte{"a.txt": []byte("x")})
	if _, err := ft.Generate(tpl, filepath.Join(tpl.Dir(), "out"), nil, nil, nil); err == nil {
		t.Fatal("generating inside the template folder must be refused")
	}
	if _, err := ft.Generate(tpl, tpl.Dir(), nil, nil, nil); err == nil {
		t.Fatal("generating into the template folder itself must be refused")
	}
}

func TestTargetAlreadyExists(t *testing.T) {
	tpl := makeTemplate(t, "e", ft.Template{Name: "e"}, map[string][]byte{"a.txt": []byte("x")})
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(target, "e"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ft.Generate(tpl, target, nil, nil, nil); err == nil {
		t.Fatal("existing target root must be refused")
	}
}

func TestContentSizeCeiling(t *testing.T) {
	tpl := makeTemplate(t, "big",
		ft.Template{Name: "big", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"huge.md.ft$": bytes.Repeat([]byte("a"), 100)})
	_, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "v"}, nil, &ft.Options{ContentSizeLimit: 10})
	if !errors.Is(err, ft.ErrContentTooLarge) {
		t.Fatalf("want ErrContentTooLarge, got %v", err)
	}
}

func TestBinarySniffRefusesNUL(t *testing.T) {
	// NB: not "nul" — that's a reserved Windows device name.
	tpl := makeTemplate(t, "nulsniff",
		ft.Template{Name: "nulsniff", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"bad.md.ft$": {'a', 0x00, 'b'}})
	_, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "v"}, nil, nil)
	if !errors.Is(err, ft.ErrBinaryContent) {
		t.Fatalf("want ErrBinaryContent, got %v", err)
	}
}

func TestBOMPreserved(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	tpl := makeTemplate(t, "bom",
		ft.Template{Name: "bom", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"doc.md.ft$": append(append([]byte{}, bom...), []byte("hi {{$p}}")...)})
	res, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "x"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(res.RootPath, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, bom) {
		t.Error("UTF-8 BOM must be preserved")
	}
	if string(out[3:]) != "hi x" {
		t.Errorf("content after BOM = %q", out[3:])
	}
}

func TestCaseOnlyCollisionDetected(t *testing.T) {
	// Source names coexist on any filesystem; renaming p→"A" makes "{p}.md"
	// collide with "a.md" on case-insensitive targets.
	tpl := makeTemplate(t, "c",
		ft.Template{Name: "c", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"a.md": []byte("1"), "{p}.md": []byte("2")})
	if _, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "A"}, nil, nil); err == nil {
		t.Fatal("case-only output collision must be detected")
	}
}

func TestRegexTimeoutIsValidationError(t *testing.T) {
	p := baseParam("p")
	p.Match = str("(a+)+$") // catastrophic backtracking on aaaa…b
	tpl := makeTemplate(t, "t",
		ft.Template{Name: "t", Parameters: []ft.Parameter{p}},
		map[string][]byte{strings.Repeat("a", 40) + "b.txt": []byte("x")})
	_, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "v"}, nil, &ft.Options{MatchTimeout: 50 * time.Millisecond})
	var verr *ft.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("want ValidationError from regex timeout, got %v", err)
	}
}

func TestSymlinkSkippedWithWarning(t *testing.T) {
	tpl := makeTemplate(t, "s", ft.Template{Name: "s"}, map[string][]byte{"real.txt": []byte("x")})
	if err := os.Symlink(filepath.Join(tpl.Dir(), "real.txt"), filepath.Join(tpl.Dir(), "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err) // Windows without developer mode
	}
	res, err := ft.Generate(tpl, t.TempDir(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(res.RootPath, "link.txt")); !os.IsNotExist(err) {
		t.Error("symlink must not be copied")
	}
	if len(res.Warnings) == 0 {
		t.Error("skipped symlink must produce a warning")
	}
}

// --- preview ---------------------------------------------------------------------

func TestPreviewMatchesGenerate(t *testing.T) {
	tpl, err := ft.Load(filepath.Join("testdata", "youtube-template"))
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"videoName": "ep7"}
	extra := map[string]string{"bruvDate": "2026-07-04"}

	entries, _, err := ft.Preview(tpl, values, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	var previewed []string
	for _, e := range entries {
		previewed = append(previewed, e.OutputRel)
	}

	res, err := ft.Generate(tpl, t.TempDir(), values, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	var actual []string
	root := filepath.Dir(res.RootPath)
	err = filepath.WalkDir(res.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		actual = append(actual, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(previewed)
	sort.Strings(actual)
	if strings.Join(previewed, "\n") != strings.Join(actual, "\n") {
		t.Errorf("preview and generate disagree:\npreview:\n%s\nactual:\n%s",
			strings.Join(previewed, "\n"), strings.Join(actual, "\n"))
	}
}

func TestRenderFile(t *testing.T) {
	tpl, err := ft.Load(filepath.Join("testdata", "youtube-template"))
	if err != nil {
		t.Fatal(err)
	}
	before, after, err := ft.RenderFile(tpl, "script.md.ft$",
		map[string]string{"videoName": "ep9"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before, "{{$videoName}}") {
		t.Error("before must be raw")
	}
	if !strings.Contains(after, "# ep9") || strings.Contains(after, "{{$videoName}}") {
		t.Errorf("after not rendered:\n%s", after)
	}
}

// AnonymousDeletionRule reproduces a real C#-authored pattern: a parameter
// with an empty name and a Match acts as a pure rename/deletion rule (here:
// stripping a "_Template - " prefix from the root folder name).
func TestAnonymousDeletionRule(t *testing.T) {
	title := baseParam("screenplayTitle")
	strip := ft.Parameter{Match: str("^_Template - "), ReplaceInFileNames: true}
	tpl := makeTemplate(t, "_Template - {screenplayTitle}",
		ft.Template{Name: "Screenplay", Parameters: []ft.Parameter{title, strip}},
		map[string][]byte{"{screenplayTitle} - Draft 01.fountain.ft$": []byte("Title: {{$screenplayTitle}}")})

	if issues := tpl.Validate(); len(issues) != 0 {
		t.Fatalf("anonymous parameter with a match must validate: %v", issues)
	}
	res, err := ft.Generate(tpl, t.TempDir(), map[string]string{"screenplayTitle": "My Film"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(res.RootPath) != "My Film" {
		t.Errorf("root = %q, want prefix stripped + token replaced", filepath.Base(res.RootPath))
	}
	if got := readOut(t, res.RootPath, "My Film - Draft 01.fountain"); got != "Title: My Film" {
		t.Errorf("content = %q", got)
	}
}

// --- validation ------------------------------------------------------------------

func TestValidateCatchesBadPatternsAndDuplicates(t *testing.T) {
	bad := ft.Template{Name: "v", Parameters: []ft.Parameter{
		{Name: "a", Match: str("(")},
		{Name: "A"},
		{Name: ""}, // anonymous without a match — useless, flagged
	}}
	issues := bad.Validate()
	if len(issues) != 3 {
		t.Fatalf("want 3 issues (bad regex, duplicate name, anonymous without match), got %d: %v", len(issues), issues)
	}
}
