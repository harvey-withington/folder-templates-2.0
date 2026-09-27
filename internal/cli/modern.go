package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	ft "github.com/harvey-withington/foldertemplate"
)

// --- argument parsing --------------------------------------------------------

type parsedArgs struct {
	pos   []string
	vals  map[string][]string
	bools map[string]bool
}

func (p parsedArgs) val(name string) string {
	if v := p.vals[name]; len(v) > 0 {
		return v[len(v)-1]
	}
	return ""
}

// parseArgs accepts --flag, --flag value, --flag=value and positionals in any
// order. valueFlags take a value; boolFlags don't; anything else is an error.
func parseArgs(args []string, valueFlags, boolFlags []string) (parsedArgs, error) {
	p := parsedArgs{vals: map[string][]string{}, bools: map[string]bool{}}
	isValue := map[string]bool{}
	for _, f := range valueFlags {
		isValue[f] = true
	}
	isBool := map[string]bool{}
	for _, f := range boolFlags {
		isBool[f] = true
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			p.pos = append(p.pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "--") || len(a) == 2 {
			p.pos = append(p.pos, a)
			continue
		}
		name, value, hasValue := strings.Cut(a[2:], "=")
		switch {
		case isBool[name]:
			if hasValue {
				return p, fmt.Errorf("--%s takes no value", name)
			}
			p.bools[name] = true
		case isValue[name]:
			if !hasValue {
				if i+1 >= len(args) {
					return p, fmt.Errorf("--%s needs a value", name)
				}
				i++
				value = args[i]
			}
			p.vals[name] = append(p.vals[name], value)
		default:
			return p, fmt.Errorf("unknown option --%s", name)
		}
	}
	return p, nil
}

func usageError(env *Env, cmd string, err error) int {
	env.errorf("ft %s: %v\nRun `ft help` for usage.\n", cmd, err)
	return ExitUsage
}

// oneTemplate parses args and requires exactly one positional <template>.
func oneTemplate(env *Env, cmd string, args, valueFlags, boolFlags []string) (parsedArgs, string, int) {
	p, err := parseArgs(args, valueFlags, boolFlags)
	if err != nil {
		return p, "", usageError(env, cmd, err)
	}
	if len(p.pos) != 1 {
		return p, "", usageError(env, cmd, errors.New("expected exactly one <template> folder"))
	}
	return p, p.pos[0], -1
}

func loadTemplate(env *Env, dir, descriptor string) (*ft.Template, int) {
	var (
		tpl *ft.Template
		err error
	)
	if descriptor != "" {
		tpl, err = ft.LoadDescriptor(descriptor, dir)
	} else {
		tpl, err = ft.Load(dir)
	}
	if err != nil {
		env.errorf("ft: %v\n", err)
		if errors.Is(err, os.ErrNotExist) {
			return nil, ExitUsage
		}
		return nil, ExitTemplate
	}
	if tpl.Parameters == nil {
		tpl.Parameters = []ft.Parameter{}
	}
	return tpl, ExitOK
}

func writeJSON(env *Env, v any) {
	enc := json.NewEncoder(env.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// --- generate ------------------------------------------------------------------

func cmdGenerate(env *Env, args []string) int {
	p, dir, code := oneTemplate(env, "generate", args,
		[]string{"target", "set", "on-conflict", "template-file"},
		[]string{"dry-run", "no-prompt", "json"})
	if code >= 0 {
		return code
	}
	policy, err := ft.ParseConflictPolicy(p.val("on-conflict"))
	if err != nil {
		return usageError(env, "generate", err)
	}
	tpl, code := loadTemplate(env, dir, p.val("template-file"))
	if tpl == nil {
		return code
	}
	if issues := tpl.Validate(); len(issues) > 0 {
		for _, e := range issues {
			env.errorf("ft: %v\n", e)
		}
		return ExitTemplate
	}

	values := map[string]string{}
	for _, kv := range p.vals["set"] {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || strings.TrimSpace(k) == "" {
			return usageError(env, "generate", fmt.Errorf("--set wants name=value, got %q", kv))
		}
		if !declares(tpl, k) {
			env.errorf("ft: warning: the template has no parameter %q\n", k)
		}
		values[k] = v
	}
	if env.Interactive && !p.bools["no-prompt"] && !p.bools["json"] {
		promptMissing(env, tpl, values)
	}

	target := p.val("target")
	if target == "" {
		target = ft.ResolveDefaultTarget(tpl)
	}

	if p.bools["dry-run"] {
		pr, err := ft.PreviewAt(tpl, target, values, nil, nil)
		if err != nil {
			env.errorf("ft: %v\n", err)
			return exitFor(err)
		}
		if p.bools["json"] {
			writeJSON(env, pr)
			return ExitOK
		}
		printPreview(env, pr, policy)
		return ExitOK
	}

	res, err := ft.GenerateContext(env.Ctx, tpl, target, values, nil, &ft.Options{Conflict: policy})
	if err != nil {
		env.errorf("ft: %v\n", err)
		if errors.Is(err, ft.ErrTargetExists) {
			env.errorf("Use --on-conflict merge to add only what is missing, or --on-conflict overwrite to replace existing files.\n")
		}
		return exitFor(err)
	}
	if p.bools["json"] {
		writeJSON(env, res)
		return ExitOK
	}
	env.printf("Created %s (%d %s)\n", res.RootPath, res.FilesWritten, plural(res.FilesWritten, "file", "files"))
	for _, s := range res.Skipped {
		env.printf("  kept existing  %s\n", s)
	}
	for _, s := range res.Overwritten {
		env.printf("  overwrote      %s\n", s)
	}
	for _, w := range res.Warnings {
		env.errorf("warning: %s\n", w)
	}
	return ExitOK
}

func declares(tpl *ft.Template, name string) bool {
	for _, p := range tpl.Parameters {
		if strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

// promptMissing asks for every prompted parameter not already answered.
// An empty answer keeps the default.
func promptMissing(env *Env, tpl *ft.Template, values map[string]string) {
	for _, p := range tpl.Parameters {
		if p.Internal() || p.Name == "" {
			continue
		}
		if _, done := lookupFold(values, p.Name); done {
			continue
		}
		prompt := *p.Prompt
		if p.DefaultValue != nil && *p.DefaultValue != "" {
			prompt += " [" + *p.DefaultValue + "]"
		} else if p.Placeholder != nil && *p.Placeholder != "" {
			prompt += " (" + *p.Placeholder + ")"
		}
		env.printf("%s: ", prompt)
		line, ok := env.readLine()
		if !ok {
			env.printf("\n")
			return
		}
		if v := strings.TrimSpace(line); v != "" {
			values[p.Name] = v
		}
	}
}

func lookupFold(m map[string]string, name string) (string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

func printPreview(env *Env, pr *ft.PreviewResult, policy ft.ConflictPolicy) {
	env.printf("Would create in %s:\n", pr.RootPath)
	conflicts := 0
	for _, e := range pr.Entries {
		depth := strings.Count(e.OutputRel, "/")
		name := e.OutputRel[strings.LastIndex(e.OutputRel, "/")+1:]
		if e.IsDir {
			name += "/"
		}
		mark := ""
		if e.Processed {
			mark += "  [filled in]"
		}
		if e.Exists {
			if !e.IsDir {
				conflicts++
			}
			mark += "  (exists)"
		}
		env.printf("  %s%s%s\n", strings.Repeat("  ", depth), name, mark)
	}
	for _, w := range pr.Warnings {
		env.printf("warning: %s\n", w)
	}
	if pr.RootExists {
		switch policy {
		case ft.ConflictRefuse:
			env.printf("\n%s already exists; generating would be refused (see --on-conflict).\n", pr.RootName)
		case ft.ConflictMerge:
			env.printf("\n%d existing %s would be kept.\n", conflicts, plural(conflicts, "file", "files"))
		case ft.ConflictOverwrite:
			env.printf("\n%d existing %s would be overwritten.\n", conflicts, plural(conflicts, "file", "files"))
		}
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// --- params / info / validate / scan -------------------------------------------

func cmdParams(env *Env, args []string) int {
	p, dir, code := oneTemplate(env, "params", args, []string{"template-file"}, []string{"all", "json"})
	if code >= 0 {
		return code
	}
	tpl, code := loadTemplate(env, dir, p.val("template-file"))
	if tpl == nil {
		return code
	}
	list := []ft.Parameter{}
	for _, prm := range tpl.Parameters {
		if p.bools["all"] || !prm.Internal() {
			list = append(list, prm)
		}
	}
	if p.bools["json"] {
		writeJSON(env, list)
		return ExitOK
	}
	if len(list) == 0 {
		env.printf("This template asks for nothing.\n")
		return ExitOK
	}
	for _, prm := range list {
		label := prm.Name
		if label == "" {
			label = "(rename rule)"
		}
		env.printf("%s\n", label)
		if prm.Prompt != nil && *prm.Prompt != "" {
			env.printf("  prompt:      %s\n", *prm.Prompt)
		} else {
			env.printf("  internal:    not asked\n")
		}
		if prm.Placeholder != nil && *prm.Placeholder != "" {
			env.printf("  placeholder: %s\n", *prm.Placeholder)
		}
		if prm.DefaultValue != nil {
			env.printf("  default:     %s\n", *prm.DefaultValue)
		}
		var where []string
		if prm.ReplaceInFileNames {
			where = append(where, "names")
		}
		if prm.ReplaceInFiles {
			where = append(where, ft.ContentExt+" content")
		}
		if len(where) == 0 {
			where = []string{"nowhere"}
		}
		env.printf("  replaces in: %s\n", strings.Join(where, ", "))
		if prm.Match != nil && *prm.Match != "" {
			env.printf("  match:       %s\n", *prm.Match)
		}
	}
	return ExitOK
}

func cmdInfo(env *Env, args []string) int {
	p, dir, code := oneTemplate(env, "info", args, nil, []string{"json"})
	if code >= 0 {
		return code
	}
	insp, err := ft.Inspect(dir)
	if err != nil {
		env.errorf("ft: %v\n", err)
		return ExitUsage
	}
	if p.bools["json"] {
		writeJSON(env, insp)
		return ExitOK
	}
	if !insp.IsTemplate {
		if insp.LoadError != "" {
			env.errorf("ft: %s\n", insp.LoadError)
			return ExitTemplate
		}
		env.errorf("ft: %s is not a template (no %s/%s)\n", insp.Dir, ft.ConfigDirName, ft.ConfigFileName)
		return ExitTemplate
	}
	t := insp.Template
	env.printf("Name:           %s\n", t.Name)
	if t.Description != "" {
		env.printf("Description:    %s\n", t.Description)
	}
	env.printf("Folder:         %s\n", insp.Dir)
	env.printf("Default target: %s\n", insp.DefaultTarget)
	env.printf("Parameters:     %d\n", len(t.Parameters))
	env.printf("Contents:       %d %s, %d %s, %s\n", insp.Files, plural(insp.Files, "file", "files"),
		insp.Dirs, plural(insp.Dirs, "folder", "folders"), humanBytes(insp.SizeBytes))
	for _, issue := range insp.Issues {
		env.printf("Issue:          %s\n", issue)
	}
	return ExitOK
}

func cmdValidate(env *Env, args []string) int {
	p, dir, code := oneTemplate(env, "validate", args, []string{"template-file"}, []string{"json"})
	if code >= 0 {
		return code
	}
	tpl, code := loadTemplate(env, dir, p.val("template-file"))
	if tpl == nil {
		return code
	}
	issues := []string{}
	for _, e := range tpl.Validate() {
		issues = append(issues, e.Error())
	}
	if p.bools["json"] {
		writeJSON(env, map[string]any{"valid": len(issues) == 0, "issues": issues})
	} else if len(issues) == 0 {
		env.printf("OK: %s\n", tpl.Name)
	} else {
		for _, s := range issues {
			env.printf("%s\n", s)
		}
	}
	if len(issues) > 0 {
		return ExitTemplate
	}
	return ExitOK
}

func cmdScan(env *Env, args []string) int {
	p, dir, code := oneTemplate(env, "scan", args, nil, []string{"json"})
	if code >= 0 {
		return code
	}
	res, err := ft.Scan(dir, nil, nil)
	if err != nil {
		env.errorf("ft: %v\n", err)
		return exitFor(err)
	}
	if p.bools["json"] {
		writeJSON(env, res)
		return ExitOK
	}
	if len(res.Tokens) == 0 {
		env.printf("No tokens found.\n")
	} else {
		env.printf("Tokens:\n")
		for _, t := range res.Tokens {
			state := "declared"
			if !t.Declared {
				state = "undeclared"
			}
			env.printf("  %-20s %-10s names: %d  content: %d\n", t.Name, state, len(t.InNames), len(t.InContent))
		}
	}
	if len(res.Issues) == 0 {
		env.printf("No issues.\n")
		return ExitOK
	}
	sort.SliceStable(res.Issues, func(i, j int) bool { return res.Issues[i].Kind < res.Issues[j].Kind })
	env.printf("Issues:\n")
	for _, is := range res.Issues {
		env.printf("  - %s\n", is.Message)
	}
	return ExitOK
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
