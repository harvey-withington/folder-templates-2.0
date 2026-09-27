package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ft "github.com/harvey-withington/foldertemplate"
)

func str(s string) *string { return &s }

// fixture writes a template folder "{name} project" with a prompted name, an
// internal channel parameter and one .ft$ file.
func fixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "{name} project")
	tpl := &ft.Template{
		Name:              "Proj",
		Description:       "A project",
		DefaultTargetPath: "Out",
		Parameters: []ft.Parameter{
			{Name: "name", Type: "text", Prompt: str("Project name?"), Placeholder: str("e.g. alpha"), ReplaceInFileNames: true, ReplaceInFiles: true},
			{Name: "owner", Type: "text", Prompt: str("Owner?"), DefaultValue: str("Harvey"), ReplaceInFiles: true},
			{Name: "channel", Type: "text", DefaultValue: str("OOP"), ReplaceInFiles: true},
		},
	}
	if err := ft.Save(tpl, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.md.ft$"), []byte("{{$name}} by {{$owner}} for {{$channel}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

type run struct {
	code   int
	stdout string
	stderr string
	waited bool
}

func runFT(t *testing.T, stdin string, interactive bool, args ...string) run {
	t.Helper()
	var out, errb bytes.Buffer
	r := run{}
	env := &Env{
		Stdin: strings.NewReader(stdin), Stdout: &out, Stderr: &errb,
		Interactive: interactive, Ctx: context.Background(), Newline: "\r\n", Version: "test",
		WaitKey: func() { r.waited = true },
	}
	r.code = Run(args, env)
	r.stdout, r.stderr = out.String(), errb.String()
	return r
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// --- v1 compatibility -------------------------------------------------------------

func TestV1ListParamsJSONIsNewtonsoftShaped(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "-sourceFolder", dir, "-listParams", "json", "-nowait")
	want := "[\r\n" +
		"  {\r\n" +
		"    \"Name\": \"name\",\r\n" +
		"    \"Type\": \"text\",\r\n" +
		"    \"Prompt\": \"Project name?\",\r\n" +
		"    \"Placeholder\": \"e.g. alpha\",\r\n" +
		"    \"DefaultValue\": null\r\n" +
		"  },\r\n" +
		"  {\r\n" +
		"    \"Name\": \"owner\",\r\n" +
		"    \"Type\": \"text\",\r\n" +
		"    \"Prompt\": \"Owner?\",\r\n" +
		"    \"Placeholder\": null,\r\n" +
		"    \"DefaultValue\": \"Harvey\"\r\n" +
		"  }\r\n" +
		"]\r\n"
	if r.code != 0 || r.stdout != want {
		t.Fatalf("code %d, stdout:\n%q\nwant:\n%q", r.code, r.stdout, want)
	}
	if r.waited {
		t.Error("-nowait must not wait for a key")
	}
}

func TestV1TemplateInfo(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "-sourceFolder", dir, "-getTemplateInfo", "json", "-nowait")
	want := "{\r\n  \"Name\": \"Proj\",\r\n  \"DefaultTargetPath\": \"Out\"\r\n}\r\n"
	if r.stdout != want {
		t.Fatalf("got %q want %q", r.stdout, want)
	}
	r = runFT(t, "", false, "-sourceFolder", dir, "-getTemplateInfo", "-nowait")
	if !strings.HasPrefix(r.stdout, "Listing Template Info (format = plain):\n\nName: Proj\nDefaultTargetPath: Out\n") {
		t.Errorf("plain info = %q", r.stdout)
	}
	r = runFT(t, "", false, "-sourceFolder", dir, "-getTemplateInfo", "xml", "-nowait")
	if r.stdout != "Invalid format: xml.\n" {
		t.Errorf("bad format = %q", r.stdout)
	}
}

func TestV1ListParamsPlain(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "-sourceFolder", dir, "-listParams", "-nowait")
	want := "Listing Template Folder params (format = plain):\n\n" +
		"name (text): 'Project name?' = []\n" +
		"owner (text): 'Owner?' = [Harvey]\n"
	if r.stdout != want {
		t.Fatalf("got %q", r.stdout)
	}
}

func TestV1GenerateWithFlagsAndPrompts(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	// -name answers the prompt; owner is prompted and answered on stdin.
	r := runFT(t, "  Alice  \n", false, "-sourceFolder", dir, "-targetFolder", target, "-name", "alpha", "-nowait")
	if r.code != 0 {
		t.Fatalf("code %d: %s", r.code, r.stdout)
	}
	if r.stdout != "Processing Template Folder...\nOwner?: Finished Processing.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
	got := read(t, filepath.Join(target, "alpha project", "readme.md"))
	if got != "alpha by Alice for OOP\n" {
		t.Errorf("readme = %q", got)
	}
}

func TestV1NoPromptUsesDefaultsAndBareSourcePath(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	r := runFT(t, "", false, dir, "-targetFolder", target, "-noprompt", "-nowait")
	if r.code != 0 {
		t.Fatalf("code %d: %s", r.code, r.stdout)
	}
	if got := read(t, filepath.Join(target, " project", "readme.md")); got != " by Harvey for OOP\n" {
		t.Errorf("readme = %q", got)
	}
}

func TestV1ParamNamesAreCaseSensitive(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	runFT(t, "", false, "-sourceFolder", dir, "-targetFolder", target, "-NAME", "loud", "-noprompt", "-nowait")
	if _, err := os.Stat(filepath.Join(target, " project")); err != nil {
		t.Error("-NAME must not answer parameter 'name' (1.0 compared names exactly)")
	}
}

func TestV1OverwritesLikeOriginal(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	args := []string{"-sourceFolder", dir, "-targetFolder", target, "-name", "x", "-noprompt", "-nowait"}
	runFT(t, "", false, args...)
	if err := os.WriteFile(filepath.Join(target, "x project", "readme.md"), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runFT(t, "", false, args...); r.code != 0 {
		t.Fatalf("second run code %d: %s", r.code, r.stdout)
	}
	if got := read(t, filepath.Join(target, "x project", "readme.md")); got != "x by Harvey for OOP\n" {
		t.Errorf("1.0 replaced existing files; got %q", got)
	}
}

func TestV1MissingPathsAndWait(t *testing.T) {
	r := runFT(t, "", false, "-sourceFolder", filepath.Join(t.TempDir(), "nope"), "-targetFolder", filepath.Join(t.TempDir(), "nope"))
	want := "The specified source path could not be found.\n" +
		"The specified target path could not be found.\n" +
		"The specified template file could not be found. Make sure the source folder has a properly-configured '.ft' subfolder, or specify the -templateFile parameter.\n" +
		"\nComplete - Press any key to exit\n"
	if r.stdout != want || r.code != ExitUsage || !r.waited {
		t.Fatalf("code %d waited %v stdout %q", r.code, r.waited, r.stdout)
	}
}

func TestV1ParseErrors(t *testing.T) {
	r := runFT(t, "", false, "-nowait")
	if !strings.Contains(r.stdout, "Syntax error of parameter -sourceFolder: Required parameter is not found.\nUse -help for more information.") || r.code != ExitUsage {
		t.Errorf("missing source: %q", r.stdout)
	}
	r = runFT(t, "", false, "-sourceFolder", "a", "-sourceFolder", "b", "-nowait")
	if !strings.Contains(r.stdout, "Syntax error of parameter -sourceFolder: Parameter is specified more than once.") {
		t.Errorf("duplicate: %q", r.stdout)
	}
}

func TestV1HelpScreen(t *testing.T) {
	r := runFT(t, "", false, "-help", "-nowait")
	if !strings.HasPrefix(r.stdout, "\nParameters:\n\n-help             Prints the help screen.\n-sourceFolder     The path of the Template Folder to process\n") {
		t.Errorf("help = %q", r.stdout)
	}
}

func TestV1TemplateFile(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "{p}.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	desc := filepath.Join(t.TempDir(), "d.json")
	if err := os.WriteFile(desc, []byte(`{"Name":"D","Parameters":[{"Name":"p","ReplaceInFileNames":true}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	r := runFT(t, "", false, "-sourceFolder", src, "-templateFile", desc, "-targetFolder", target, "-p", "v", "-nowait")
	if r.code != 0 {
		t.Fatalf("code %d: %s", r.code, r.stdout)
	}
	if _, err := os.Stat(filepath.Join(target, filepath.Base(src), "v.txt")); err != nil {
		t.Error("-templateFile descriptor was not applied")
	}
}

// --- modern commands ----------------------------------------------------------------

func TestGenerateWithSetAndDefaultTarget(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "generate", dir, "--set", "name=beta", "--set=owner=Bob")
	if r.code != 0 {
		t.Fatalf("code %d: %s", r.code, r.stderr)
	}
	root := filepath.Join(filepath.Dir(dir), "Out", "beta project")
	if got := read(t, filepath.Join(root, "readme.md")); got != "beta by Bob for OOP\n" {
		t.Errorf("readme = %q", got)
	}
	if !strings.Contains(r.stdout, "Created "+root+" (1 file)") {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestGenerateConflictExitCodeAndPolicies(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	args := []string{"generate", dir, "--target", target, "--set", "name=c", "--no-prompt"}
	if r := runFT(t, "", false, args...); r.code != 0 {
		t.Fatalf("first run: %d %s", r.code, r.stderr)
	}
	r := runFT(t, "", false, args...)
	if r.code != ExitConflict || !strings.Contains(r.stderr, "--on-conflict") {
		t.Fatalf("second run: code %d stderr %q", r.code, r.stderr)
	}
	r = runFT(t, "", false, append(args, "--on-conflict", "merge")...)
	if r.code != 0 || !strings.Contains(r.stdout, "kept existing  readme.md") {
		t.Errorf("merge: %d %q", r.code, r.stdout)
	}
	r = runFT(t, "", false, append(args, "--on-conflict", "bogus")...)
	if r.code != ExitUsage {
		t.Errorf("bad policy code %d", r.code)
	}
}

func TestGenerateDryRunWritesNothing(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	r := runFT(t, "", false, "generate", dir, "--target", target, "--set", "name=d", "--dry-run")
	if r.code != 0 || !strings.Contains(r.stdout, "d project/") || !strings.Contains(r.stdout, "readme.md  [filled in]") {
		t.Fatalf("dry run: %d %q", r.code, r.stdout)
	}
	entries, _ := os.ReadDir(target)
	if len(entries) != 0 {
		t.Error("dry run wrote to the target")
	}
	r = runFT(t, "", false, "generate", dir, "--target", target, "--set", "name=d", "--dry-run", "--json")
	var pr ft.PreviewResult
	if err := json.Unmarshal([]byte(r.stdout), &pr); err != nil || pr.RootName != "d project" {
		t.Errorf("dry run json: %v %+v", err, pr)
	}
}

func TestGeneratePromptsWhenInteractive(t *testing.T) {
	dir := fixture(t)
	target := t.TempDir()
	r := runFT(t, "gamma\n\n", true, "generate", dir, "--target", target)
	if r.code != 0 {
		t.Fatalf("code %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "Project name? (e.g. alpha): Owner? [Harvey]: ") {
		t.Errorf("prompts = %q", r.stdout)
	}
	if got := read(t, filepath.Join(target, "gamma project", "readme.md")); got != "gamma by Harvey for OOP\n" {
		t.Errorf("readme = %q", got)
	}
}

func TestGenerateWarnsOnUnknownSet(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "generate", dir, "--target", t.TempDir(), "--set", "nme=typo", "--no-prompt")
	if !strings.Contains(r.stderr, `no parameter "nme"`) {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestGenerateRejectsInvalidTemplate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bad")
	if err := ft.Save(&ft.Template{Name: "bad", Parameters: []ft.Parameter{{Name: "x", Match: str("("), ReplaceInFileNames: true}}}, dir); err != nil {
		t.Fatal(err)
	}
	if r := runFT(t, "", false, "generate", dir, "--target", t.TempDir()); r.code != ExitTemplate {
		t.Errorf("code %d", r.code)
	}
	if r := runFT(t, "", false, "validate", dir); r.code != ExitTemplate || !strings.Contains(r.stdout, "invalid match pattern") {
		t.Errorf("validate: %d %q", r.code, r.stdout)
	}
}

func TestParamsAndInfoAndScan(t *testing.T) {
	dir := fixture(t)
	r := runFT(t, "", false, "params", dir, "--json")
	var params []ft.Parameter
	if err := json.Unmarshal([]byte(r.stdout), &params); err != nil || len(params) != 2 {
		t.Errorf("params --json: %v %d", err, len(params))
	}
	r = runFT(t, "", false, "params", dir, "--all", "--json")
	if err := json.Unmarshal([]byte(r.stdout), &params); err != nil || len(params) != 3 {
		t.Errorf("params --all --json: %v %d", err, len(params))
	}
	r = runFT(t, "", false, "info", dir)
	if r.code != 0 || !strings.Contains(r.stdout, "Name:           Proj") || !strings.Contains(r.stdout, "Parameters:     3") {
		t.Errorf("info: %q", r.stdout)
	}
	r = runFT(t, "", false, "validate", dir, "--json")
	if !strings.Contains(r.stdout, `"valid": true`) {
		t.Errorf("validate --json: %q", r.stdout)
	}
	r = runFT(t, "", false, "scan", dir)
	if r.code != 0 || !strings.Contains(r.stdout, "name") || !strings.Contains(r.stdout, "No issues.") {
		t.Errorf("scan: %q", r.stdout)
	}
}

func TestNotATemplate(t *testing.T) {
	plain := t.TempDir()
	if r := runFT(t, "", false, "params", plain); r.code != ExitTemplate {
		t.Errorf("params on plain folder: code %d", r.code)
	}
	if r := runFT(t, "", false, "info", plain); r.code != ExitTemplate {
		t.Errorf("info on plain folder: code %d", r.code)
	}
}

func TestUsageAndUnknownOptions(t *testing.T) {
	if r := runFT(t, "", false); r.code != ExitUsage || !strings.Contains(r.stdout, "Usage:") {
		t.Errorf("no args: %d", r.code)
	}
	if r := runFT(t, "", false, "help"); r.code != 0 {
		t.Errorf("help: %d", r.code)
	}
	if r := runFT(t, "", false, "--bogus"); r.code != ExitUsage {
		t.Errorf("--bogus: %d", r.code)
	}
	if r := runFT(t, "", false, "generate", "a", "--wat"); r.code != ExitUsage || !strings.Contains(r.stderr, "unknown option --wat") {
		t.Errorf("unknown flag: %d %q", r.code, r.stderr)
	}
	if r := runFT(t, "", false, "version"); r.stdout != "ft test\n" {
		t.Errorf("version: %q", r.stdout)
	}
}

func TestNetStringEscaping(t *testing.T) {
	bs, q := string(rune(92)), string(rune(34))
	ls := string(rune(0x2028))
	in := "a" + q + "b" + bs + "c" + string(rune(10)) + string(rune(9)) + string(rune(1)) + "é<>&" + ls
	want := q + "a" + bs + q + "b" + bs + bs + "c" + bs + "n" + bs + "t" + bs + "u0001" + "é<>&" + bs + "u2028" + q
	if got := netString(in); got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
