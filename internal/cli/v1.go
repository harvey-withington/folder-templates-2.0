package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	ft "github.com/harvey-withington/folder-templates-2.0/engine"
)

// v1 compatibility: the C# FolderTemplates.Console command line, reproduced
// down to its parser quirks and messages so existing scripts, SendTo
// shortcuts and the Obsidian plugin keep working against ft.exe.

type v1Param struct {
	name       string
	help       string
	required   bool
	registered bool
	exists     bool
	value      *string
}

type v1Line struct {
	byName map[string]*v1Param // case-sensitive, like the C# Dictionary
	order  []string
}

func newV1Line() *v1Line {
	l := &v1Line{byName: map[string]*v1Param{}}
	for _, p := range []v1Param{
		{name: "help", help: "Prints the help screen."},
		{name: "sourceFolder", required: true, help: "The path of the Template Folder to process"},
		{name: "templateFile", help: "The path of the Template Folder Definition file to apply"},
		{name: "targetFolder", help: "The path of the folder in which to generate the template result"},
		{name: "listParams", help: "Don't process the template folder, just list its parameters"},
		{name: "getTemplateInfo", help: "Don't process the template folder, just list its properties"},
		{name: "nowait", help: "Close the console after processing, do not wait for keypress"},
		{name: "noprompt", help: "Do not prompt for missing parameters, use defaults instead"},
	} {
		p := p
		p.registered = true
		l.add(&p)
	}
	return l
}

func (l *v1Line) add(p *v1Param) {
	l.byName[p.name] = p
	l.order = append(l.order, p.name)
}

func (l *v1Line) get(name string) *v1Param { return l.byName[name] }

// parse mirrors CommandLineProcessor.Parse(args, allowUnspecified: true,
// defaultUnspecifiedFlag: "sourceFolder"): "-key [value]" pairs, where the
// next argument is the value unless it starts with "-"; bare arguments set
// -sourceFolder; unknown keys are kept as template parameter values.
func (l *v1Line) parse(args []string) error {
	for i := 0; i < len(args); {
		a := args[i]
		if len(a) > 1 && a[0] == '-' {
			key := a[1:]
			var value *string
			i++
			if i < len(args) && !(len(args[i]) > 0 && args[i][0] == '-') {
				v := args[i]
				value = &v
				i++
			}
			if p, ok := l.byName[key]; ok {
				if p.exists {
					return fmt.Errorf("Syntax error of parameter -%s: Parameter is specified more than once.", key)
				}
				p.exists, p.value = true, value
			} else {
				l.add(&v1Param{name: key, exists: true, value: value})
			}
			continue
		}
		v := a
		src := l.get("sourceFolder")
		src.exists, src.value = true, &v
		i++
	}
	for _, name := range l.order {
		if p := l.byName[name]; p.required && !p.exists {
			return fmt.Errorf("Syntax error of parameter -%s: Required parameter is not found.", name)
		}
	}
	return nil
}

func (l *v1Line) helpScreen() string {
	width := 0
	for _, name := range l.order {
		width = max(width, len(name))
	}
	var b strings.Builder
	b.WriteString("\nParameters:\n\n")
	for _, name := range l.order {
		s := "-" + name
		for len(s) < width+3 {
			s += " "
		}
		b.WriteString(s + l.byName[name].help + "\n")
	}
	return b.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func runV1(env *Env, args []string) int {
	l := newV1Line()
	perr := l.parse(args)
	if l.get("help").exists {
		env.printf("%s\n", l.helpScreen())
	}
	code := ExitOK
	if perr != nil {
		env.printf("%s\nUse -help for more information.\n", perr)
		code = ExitUsage
	} else {
		code = runV1Parsed(env, l)
	}
	if !l.get("nowait").exists {
		env.printf("\nComplete - Press any key to exit\n")
		if env.WaitKey != nil {
			env.WaitKey()
		}
	}
	return code
}

func runV1Parsed(env *Env, l *v1Line) int {
	srcArg := deref(l.get("sourceFolder").value)
	sourcePath, _ := filepath.Abs(srcArg)
	targetArg := filepath.Dir(srcArg)
	if t := l.get("targetFolder"); t.value != nil {
		targetArg = *t.value
	}
	targetPath, _ := filepath.Abs(targetArg)
	templateArg := filepath.Join(srcArg, ft.ConfigDirName, ft.ConfigFileName)
	if t := l.get("templateFile"); t.value != nil {
		templateArg = *t.value
	}
	templateFile, _ := filepath.Abs(templateArg)

	failed := false
	if !isDir(sourcePath) {
		env.printf("The specified source path could not be found.\n")
		failed = true
	}
	if !isDir(targetPath) {
		env.printf("The specified target path could not be found.\n")
		failed = true
	}
	if !isFile(templateFile) {
		env.printf("The specified template file could not be found. Make sure the source folder has a properly-configured '.ft' subfolder, or specify the -templateFile parameter.\n")
		failed = true
	}
	if failed {
		return ExitUsage
	}

	tpl, err := ft.LoadDescriptor(templateFile, sourcePath)
	if err != nil {
		env.printf("Could not load template file: %v\n", err)
		return ExitTemplate
	}

	if p := l.get("listParams"); p.exists {
		return v1ListParams(env, tpl, formatOf(p))
	}
	if p := l.get("getTemplateInfo"); p.exists {
		return v1TemplateInfo(env, tpl, formatOf(p))
	}

	env.printf("Processing Template Folder...\n")
	// Unregistered flags answer parameters whose name matches exactly
	// (the C# app compared names case-sensitively).
	answers := map[string]*string{}
	for _, name := range l.order {
		p := l.byName[name]
		if p.registered {
			continue
		}
		for _, tp := range tpl.Parameters {
			if tp.Name == name {
				answers[tp.Name] = p.value
				break
			}
		}
	}
	if !l.get("noprompt").exists {
		for _, tp := range tpl.Parameters {
			if answers[tp.Name] != nil || tp.Prompt == nil {
				continue
			}
			env.printf("%s: ", *tp.Prompt)
			line, ok := env.readLine()
			if ok && strings.TrimSpace(line) != "" {
				v := strings.TrimSpace(line)
				answers[tp.Name] = &v
			}
		}
	}
	values := map[string]string{}
	for name, v := range answers {
		if v != nil {
			values[name] = *v
		}
	}
	// The C# console merged into an existing folder and replaced files.
	_, err = ft.GenerateContext(env.Ctx, tpl, targetPath, values, nil, &ft.Options{Conflict: ft.ConflictOverwrite})
	if err != nil {
		env.printf("Error: %v\n", err)
		return exitFor(err)
	}
	env.printf("Finished Processing.\n")
	return ExitOK
}

func formatOf(p *v1Param) string {
	if p.value == nil || strings.TrimSpace(*p.value) == "" {
		return "plain"
	}
	return *p.value
}

func v1ListParams(env *Env, tpl *ft.Template, format string) int {
	// Public parameters are those with a prompt — including an empty one,
	// which the C# app (testing Prompt != null) still listed.
	var public []ft.Parameter
	for _, p := range tpl.Parameters {
		if p.Prompt != nil {
			public = append(public, p)
		}
	}
	typeOf := func(p ft.Parameter) string {
		if p.Type == "" {
			return "text" // ParameterInfo's default when the key is absent
		}
		return p.Type
	}
	switch format {
	case "plain":
		env.printf("Listing Template Folder params (format = plain):\n\n")
		for _, p := range public {
			env.printf("%s (%s): '%s' = [%s]\n", p.Name, typeOf(p), deref(p.Prompt), deref(p.DefaultValue))
		}
	case "json":
		objects := make([][]netField, 0, len(public))
		for _, p := range public {
			name, typ := p.Name, typeOf(p)
			objects = append(objects, []netField{
				{"Name", &name}, {"Type", &typ}, {"Prompt", p.Prompt},
				{"Placeholder", p.Placeholder}, {"DefaultValue", p.DefaultValue},
			})
		}
		env.printf("%s%s", netArray(objects, env.Newline), env.Newline)
	default:
		env.printf("Invalid format: %s.\n", format)
	}
	return ExitOK
}

func v1TemplateInfo(env *Env, tpl *ft.Template, format string) int {
	nullable := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	switch format {
	case "plain":
		env.printf("Listing Template Info (format = plain):\n\n")
		env.printf("Name: %s\n", tpl.Name)
		env.printf("DefaultTargetPath: %s\n", tpl.DefaultTargetPath)
	case "json":
		fields := []netField{{"Name", nullable(tpl.Name)}, {"DefaultTargetPath", nullable(tpl.DefaultTargetPath)}}
		env.printf("%s%s", netObject(fields, "", env.Newline), env.Newline)
	default:
		env.printf("Invalid format: %s.\n", format)
	}
	return ExitOK
}
