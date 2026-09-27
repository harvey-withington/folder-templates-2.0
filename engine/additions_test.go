package foldertemplate_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"

	ft "github.com/harvey-withington/foldertemplate"
)

// --- conflict policy -------------------------------------------------------------

// existingRoot generates the template once, then edits the output so a second
// run has something to merge into or overwrite.
func existingRoot(t *testing.T) (*ft.Template, string) {
	t.Helper()
	tpl := makeTemplate(t, "{p}",
		ft.Template{Name: "c", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"a.txt": []byte("new-a"), "sub/b.txt": []byte("new-b")})
	target := t.TempDir()
	root := filepath.Join(target, "X")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	return tpl, target
}

func TestConflictRefuseIsDefault(t *testing.T) {
	tpl, target := existingRoot(t)
	_, err := ft.Generate(tpl, target, map[string]string{"p": "X"}, nil, nil)
	if !errors.Is(err, ft.ErrTargetExists) {
		t.Fatalf("err = %v, want ErrTargetExists", err)
	}
	if got := readOut(t, filepath.Join(target, "X"), "a.txt"); got != "old-a" {
		t.Errorf("refuse must not touch existing files, a.txt = %q", got)
	}
}

func TestConflictMergeSkipsExisting(t *testing.T) {
	tpl, target := existingRoot(t)
	res, err := ft.Generate(tpl, target, map[string]string{"p": "X"}, nil, &ft.Options{Conflict: ft.ConflictMerge})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(target, "X")
	if got := readOut(t, root, "a.txt"); got != "old-a" {
		t.Errorf("merge must keep existing a.txt, got %q", got)
	}
	if got := readOut(t, root, "sub/b.txt"); got != "new-b" {
		t.Errorf("merge must create missing sub/b.txt, got %q", got)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "a.txt" || res.FilesWritten != 1 {
		t.Errorf("result = %+v, want a.txt skipped and 1 file written", res)
	}
}

func TestConflictOverwriteReplaces(t *testing.T) {
	tpl, target := existingRoot(t)
	res, err := ft.Generate(tpl, target, map[string]string{"p": "X"}, nil, &ft.Options{Conflict: ft.ConflictOverwrite})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(target, "X")
	if got := readOut(t, root, "a.txt"); got != "new-a" {
		t.Errorf("overwrite must replace a.txt, got %q", got)
	}
	if len(res.Overwritten) != 1 || res.Overwritten[0] != "a.txt" || res.FilesWritten != 2 {
		t.Errorf("result = %+v, want a.txt overwritten and 2 files written", res)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt.ft-partial")); !os.IsNotExist(err) {
		t.Error("overwrite left its temp file behind")
	}
}

func TestOverwriteFailureKeepsOriginal(t *testing.T) {
	tpl := makeTemplate(t, "o",
		ft.Template{Name: "o", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"bad.txt.ft$": []byte("bin\x00ary")})
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(target, "o"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "o", "bad.txt"), []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ft.Generate(tpl, target, nil, nil, &ft.Options{Conflict: ft.ConflictOverwrite})
	if !errors.Is(err, ft.ErrBinaryContent) {
		t.Fatalf("err = %v, want ErrBinaryContent", err)
	}
	if got := readOut(t, filepath.Join(target, "o"), "bad.txt"); got != "precious" {
		t.Errorf("a failed overwrite destroyed the original: %q", got)
	}
}

func TestParseConflictPolicy(t *testing.T) {
	for in, want := range map[string]ft.ConflictPolicy{"": ft.ConflictRefuse, "Merge": ft.ConflictMerge, " overwrite ": ft.ConflictOverwrite} {
		got, err := ft.ParseConflictPolicy(in)
		if err != nil || got != want {
			t.Errorf("ParseConflictPolicy(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ft.ParseConflictPolicy("clobber"); err == nil {
		t.Error("unknown policy must error")
	}
}

// --- progress & cancellation --------------------------------------------------

func TestProgressReportsEveryEntry(t *testing.T) {
	tpl := makeTemplate(t, "g", ft.Template{Name: "g"},
		map[string][]byte{"a.txt": nil, "d/b.txt": nil, "d/c.txt": nil})
	var calls []int
	total := 0
	_, err := ft.Generate(tpl, t.TempDir(), nil, nil, &ft.Options{Progress: func(done, n int, _ string) {
		calls = append(calls, done)
		total = n
	}})
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(calls) != 4 || calls[3] != 4 {
		t.Errorf("progress calls = %v (total %d), want 1..4 of 4", calls, total)
	}
}

func TestGenerateContextCancelled(t *testing.T) {
	tpl := makeTemplate(t, "g", ft.Template{Name: "g"}, map[string][]byte{"a.txt": nil})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ft.GenerateContext(ctx, tpl, t.TempDir(), nil, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// --- preview at target & default target ------------------------------------------

func TestPreviewAtMarksExisting(t *testing.T) {
	tpl, target := existingRoot(t)
	res, err := ft.PreviewAt(tpl, target, map[string]string{"p": "X"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.RootExists || res.RootName != "X" {
		t.Fatalf("root = %q exists=%v", res.RootName, res.RootExists)
	}
	exists := map[string]bool{}
	for _, e := range res.Entries {
		exists[e.OutputRel] = e.Exists
	}
	if !exists["X/a.txt"] || !exists["X/sub"] || exists["X/sub/b.txt"] {
		t.Errorf("exists = %v, want a.txt and sub existing, sub/b.txt new", exists)
	}
}

func TestPreviewAtAppliesRecursionGuard(t *testing.T) {
	tpl := makeTemplate(t, "g", ft.Template{Name: "g"}, nil)
	if _, err := ft.PreviewAt(tpl, filepath.Join(tpl.Dir(), "inner"), nil, nil, nil); err == nil {
		t.Fatal("previewing into the template itself must fail like Generate")
	}
}

func TestResolveDefaultTarget(t *testing.T) {
	tpl := makeTemplate(t, "tpl", ft.Template{Name: "t"}, nil)
	parent := filepath.Dir(tpl.Dir())
	abs := filepath.Join(t.TempDir(), "elsewhere")
	cases := map[string]string{
		"":             parent,
		"Episodes":     filepath.Join(parent, "Episodes"),
		"../Up":        filepath.Join(filepath.Dir(parent), "Up"),
		abs:            abs,
		"sub/../Other": filepath.Join(parent, "Other"),
	}
	for dtp, want := range cases {
		tpl.DefaultTargetPath = dtp
		if got := ft.ResolveDefaultTarget(tpl); got != want {
			t.Errorf("DefaultTargetPath %q → %q, want %q", dtp, got, want)
		}
	}
}

// --- hidden .ft -----------------------------------------------------------------

func TestSaveHidesConfigDirOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("hidden attribute is Windows-only")
	}
	dir := filepath.Join(t.TempDir(), "h")
	if err := ft.Save(&ft.Template{Name: "h"}, dir); err != nil {
		t.Fatal(err)
	}
	p, _ := syscall.UTF16PtrFromString(filepath.Join(dir, ".ft"))
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		t.Fatal(err)
	}
	if attrs&syscall.FILE_ATTRIBUTE_HIDDEN == 0 {
		t.Error(".ft was not marked hidden")
	}
}

// --- inspect / tree / content toggle -------------------------------------------------

func TestInspectTemplateAndPlainFolder(t *testing.T) {
	tpl := makeTemplate(t, "i",
		ft.Template{Name: "Insp", DefaultTargetPath: "Out", Parameters: []ft.Parameter{baseParam("p"), baseParam("P")}},
		map[string][]byte{"a.txt": []byte("12345"), "d/": nil})
	insp, err := ft.Inspect(tpl.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if !insp.IsTemplate || insp.Template.Name != "Insp" || insp.Files != 1 || insp.Dirs != 1 || insp.SizeBytes != 5 {
		t.Errorf("inspection = %+v", insp)
	}
	if len(insp.Issues) != 1 || !strings.Contains(insp.Issues[0], "duplicate") {
		t.Errorf("issues = %v, want the duplicate-name issue", insp.Issues)
	}
	if insp.DefaultTarget != filepath.Join(filepath.Dir(tpl.Dir()), "Out") {
		t.Errorf("default target = %q", insp.DefaultTarget)
	}

	plain := t.TempDir()
	insp, err = ft.Inspect(plain)
	if err != nil {
		t.Fatal(err)
	}
	if insp.IsTemplate || insp.LoadError != "" || insp.Template != nil {
		t.Errorf("plain folder inspection = %+v", insp)
	}
}

func TestInspectReportsBrokenDescriptor(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ft"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ft", "template.json"), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	insp, err := ft.Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if insp.IsTemplate || insp.LoadError == "" {
		t.Errorf("broken descriptor must surface as LoadError, got %+v", insp)
	}
}

func TestTreeExcludesConfigDir(t *testing.T) {
	tpl := makeTemplate(t, "tr", ft.Template{Name: "tr"},
		map[string][]byte{"b.md.ft$": []byte("x"), "a/": nil})
	entries, err := ft.Tree(tpl.Dir())
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, e := range entries {
		rels = append(rels, e.Rel)
		if strings.HasPrefix(e.Rel, ".ft") {
			t.Errorf("tree leaked config dir entry %q", e.Rel)
		}
		if e.Rel == "b.md.ft$" && !e.Processed {
			t.Error("b.md.ft$ should be marked processed")
		}
	}
	if strings.Join(rels, ",") != "a,b.md.ft$" {
		t.Errorf("tree = %v", rels)
	}
}

func TestSetContentProcessing(t *testing.T) {
	tpl := makeTemplate(t, "s", ft.Template{Name: "s"},
		map[string][]byte{"d/readme.md": []byte("x"), "d/other.md.ft$": []byte("y"), "d/other.md": []byte("clash")})
	dir := tpl.Dir()

	got, err := ft.SetContentProcessing(dir, "d/readme.md", true)
	if err != nil || got != "d/readme.md.ft$" {
		t.Fatalf("on: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "d", "readme.md.ft$")); err != nil {
		t.Error("file was not renamed")
	}
	got, err = ft.SetContentProcessing(dir, "d/readme.md.ft$", true)
	if err != nil || got != "d/readme.md.ft$" {
		t.Errorf("already on must be a no-op: %q, %v", got, err)
	}
	got, err = ft.SetContentProcessing(dir, "d/readme.md.ft$", false)
	if err != nil || got != "d/readme.md" {
		t.Errorf("off: %q, %v", got, err)
	}
	if _, err := ft.SetContentProcessing(dir, "d/other.md.ft$", false); err == nil {
		t.Error("stripping onto an existing file must be refused")
	}
	for _, bad := range []string{"../x", ".ft/template.json", "", "d"} {
		if _, err := ft.SetContentProcessing(dir, bad, true); err == nil {
			t.Errorf("rel %q must be rejected", bad)
		}
	}
}

// --- scan & match tester --------------------------------------------------------------

func TestScanFindsTokensAndMistakes(t *testing.T) {
	names := baseParam("name")
	namesOnly := baseParam("date")
	namesOnly.ReplaceInFiles = false
	unused := baseParam("ghost")
	tpl := makeTemplate(t, "{name} project",
		ft.Template{Name: "scan", Parameters: []ft.Parameter{names, namesOnly, unused}},
		map[string][]byte{
			"{date} notes.md":    []byte("plain {{$name}} here"), // token in a non-.ft$ file
			"script.md.ft$":      []byte("Hi {{$NAME}} on {{$date}} by {{$author}}"),
			"{client}/brief.txt": []byte("x"),
			"logo.png":           []byte("\x89PNG\x00{{$name}}"), // binary: ignored
		})
	res, err := ft.Scan(tpl.Dir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	tokens := map[string]ft.TokenInfo{}
	for _, ti := range res.Tokens {
		tokens[strings.ToLower(ti.Name)] = ti
	}
	if ti := tokens["name"]; !ti.Declared || len(ti.InNames) != 1 || ti.InNames[0] != "." || len(ti.InContent) != 1 {
		t.Errorf("name token = %+v", ti)
	}
	if ti := tokens["client"]; ti.Declared || len(ti.InNames) != 1 {
		t.Errorf("client token = %+v", ti)
	}
	if ti := tokens["author"]; ti.Declared || len(ti.InContent) != 1 {
		t.Errorf("author token = %+v", ti)
	}

	kinds := map[string][]string{}
	for _, is := range res.Issues {
		kinds[is.Kind] = append(kinds[is.Kind], is.Token+is.Param+is.Path)
	}
	for k := range kinds {
		sort.Strings(kinds[k])
	}
	want := map[string][]string{
		ft.IssueTokenInUnprocessedFile: {"{{$name}}{date} notes.md"},
		ft.IssueUndeclaredToken:        {"author", "client"},
		ft.IssueUnusedParameter:        {"ghost"},
		ft.IssueContentReplacementOff:  {"datedate"},
	}
	for k, v := range want {
		if strings.Join(kinds[k], "|") != strings.Join(v, "|") {
			t.Errorf("issues[%s] = %v, want %v", k, kinds[k], v)
		}
	}
	if len(kinds[ft.IssueUnprocessableContent]) != 0 {
		t.Errorf("unexpected unprocessable issues: %v", kinds[ft.IssueUnprocessableContent])
	}

	usage := map[string]ft.ParamUsage{}
	for _, u := range res.Params {
		usage[u.Name] = u
	}
	if u := usage["name"]; u.NameHits != 1 || u.ContentHits != 1 {
		t.Errorf("name usage = %+v", u)
	}
}

func TestScanUsesUnsavedDescriptor(t *testing.T) {
	tpl := makeTemplate(t, "u", ft.Template{Name: "u"}, map[string][]byte{"{x}.txt": nil})
	res, err := ft.Scan(tpl.Dir(), &ft.Template{Parameters: []ft.Parameter{baseParam("x")}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tokens) != 1 || !res.Tokens[0].Declared || len(res.Issues) != 0 {
		t.Errorf("scan with editor descriptor = %+v", res)
	}
}

func TestScanCustomMatchCoversNameToken(t *testing.T) {
	year := baseParam("year")
	year.Match = str(`\{yyyy\}`)
	tpl := makeTemplate(t, "c", ft.Template{Name: "c", Parameters: []ft.Parameter{year}},
		map[string][]byte{"report {yyyy}.txt": nil})
	res, err := ft.Scan(tpl.Dir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tokens) != 1 || !res.Tokens[0].Declared || len(res.Issues) != 0 {
		t.Errorf("{yyyy} covered by year's match should be declared with no issues: %+v", res)
	}
}

func TestScanPlainFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "{who}.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := ft.Scan(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tokens) != 1 || res.Tokens[0].Declared {
		t.Errorf("plain folder scan = %+v", res)
	}
}

func TestScanFlagsBinaryFtFile(t *testing.T) {
	tpl := makeTemplate(t, "b", ft.Template{Name: "b"}, map[string][]byte{"img.png.ft$": []byte("\x00\x01")})
	res, err := ft.Scan(tpl.Dir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Issues) != 1 || res.Issues[0].Kind != ft.IssueUnprocessableContent {
		t.Errorf("issues = %+v", res.Issues)
	}
}

func TestTestMatch(t *testing.T) {
	tpl := makeTemplate(t, "_Template - {ep}", ft.Template{Name: "m"},
		map[string][]byte{"{ep} script.md": nil, "keep.txt": nil, "_Template - notes/": nil})
	hits, err := ft.TestMatch(tpl.Dir(), `^_Template - `, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, h := range hits {
		got[h.SourceRel] = h.Result
	}
	if len(got) != 2 || got["."] != "{ep}" || got["_Template - notes"] != "notes" {
		t.Errorf("hits = %+v", hits)
	}

	hits, err = ft.TestMatch(tpl.Dir(), `\{(\w+)\}`, "<$1>", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Errorf("backreference hits = %+v", hits)
	}
	for _, h := range hits {
		if h.SourceRel == "{ep} script.md" && h.Result != "<ep> script.md" {
			t.Errorf(".NET-style $1 substitution: got %q", h.Result)
		}
	}

	if _, err := ft.TestMatch(tpl.Dir(), `(`, "", nil); err == nil {
		t.Error("invalid pattern must error")
	}
	if _, err := ft.TestMatch(tpl.Dir(), "", "", nil); err == nil {
		t.Error("empty pattern must error")
	}
}

func TestLoadDescriptorFromElsewhere(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "{p}.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "custom.json")
	if err := os.WriteFile(file, []byte(`{"Name":"Ext","Parameters":[{"Name":"p","ReplaceInFileNames":true}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl, err := ft.LoadDescriptor(file, src)
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "Ext" || tpl.Dir() != src {
		t.Fatalf("descriptor = %+v dir %q", tpl, tpl.Dir())
	}
	res, err := ft.Generate(tpl, t.TempDir(), map[string]string{"p": "v"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(res.RootPath, "v.txt")); err != nil {
		t.Error("external descriptor was not applied")
	}
}

func TestBindUsesUnsavedDescriptor(t *testing.T) {
	tpl := makeTemplate(t, "b", ft.Template{Name: "saved"}, map[string][]byte{"{x}.txt": nil})
	bound, err := ft.Bind(ft.Template{Name: "draft", Parameters: []ft.Parameter{baseParam("x")}}, tpl.Dir())
	if err != nil {
		t.Fatal(err)
	}
	entries, _, err := ft.Preview(bound, map[string]string{"x": "v"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if entries[1].OutputRel != "b/v.txt" {
		t.Errorf("preview with bound draft = %+v", entries)
	}
	if reloaded, _ := ft.Load(tpl.Dir()); reloaded.Name != "saved" {
		t.Error("Bind must not touch the saved descriptor")
	}
}

func TestValuesCannotEscapeTheTarget(t *testing.T) {
	tpl := makeTemplate(t, "{p}", ft.Template{Name: "e", Parameters: []ft.Parameter{baseParam("p")}},
		map[string][]byte{"{p}.txt": []byte("x")})
	target := t.TempDir()
	for _, v := range []string{"..", "../escape", `..\escape`, "a/b"} {
		if _, err := ft.Generate(tpl, target, map[string]string{"p": v}, nil, nil); !errors.Is(err, ft.ErrInvalidName) {
			t.Errorf("value %q: err = %v, want ErrInvalidName", v, err)
		}
		if _, _, err := ft.Preview(tpl, map[string]string{"p": v}, nil, nil); !errors.Is(err, ft.ErrInvalidName) {
			t.Errorf("preview with value %q should fail the same way, got %v", v, err)
		}
	}
	if entries, _ := os.ReadDir(filepath.Dir(target)); len(entries) == 0 {
		t.Fatal("sanity: temp dir listing failed")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), "escape")); err == nil {
		t.Fatal("a value escaped the target folder")
	}
}

func TestWindowsInvalidCharacterReportedByPreview(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows naming rules")
	}
	tpl := makeTemplate(t, "{title}", ft.Template{Name: "w", Parameters: []ft.Parameter{baseParam("title")}}, nil)
	_, _, err := ft.Preview(tpl, map[string]string{"title": "Why is the sky?"}, nil, nil)
	if !errors.Is(err, ft.ErrInvalidName) || !strings.Contains(err.Error(), `"?"`) {
		t.Errorf("preview should explain the bad character, got %v", err)
	}
}
