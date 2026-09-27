// Package launch turns the app's command line into what the window should
// open. It accepts the Folder Templates 1.0 GUI syntax (the SendTo shortcuts
// append the selected folders to "-edit -sourceFolder") as well as the v2
// flags used by the Explorer context menu.
package launch

import "strings"

// Mode is the view the app opens in.
type Mode string

const (
	// ModeHome shows the template library.
	ModeHome Mode = "home"
	// ModeOpen opens Source: the generate view for a template, or the
	// "make this a template" editor for a plain folder (decided by the app
	// after inspecting Source, as 1.0 did).
	ModeOpen Mode = "open"
	// ModeEdit opens Source in the template editor.
	ModeEdit Mode = "edit"
	// ModePick shows the library as a picker with the target fixed — the
	// "New from template here…" context menu.
	ModePick Mode = "pick"
)

// Intent is the parsed command line.
type Intent struct {
	Mode   Mode   `json:"mode"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	// Extra holds further folders selected alongside Source (SendTo with a
	// multi-selection); each gets its own window.
	Extra []string `json:"extra,omitempty"`
}

// Parse reads args (without the program name).
//
// Accepted: a bare folder, -sourceFolder/--source <dir>, -targetFolder/
// --target <dir>, -edit/--edit (a flag; the folder comes from -sourceFolder or
// a bare argument, as 1.0's "Edit" SendTo shortcut passes it), and
// --pick-template. Flag names match case-insensitively; unknown flags are
// ignored, as 1.0 ignored them.
func Parse(args []string) Intent {
	var (
		sources []string
		target  string
		edit    bool
		pick    bool
	)
	takesValue := map[string]bool{"sourcefolder": true, "source": true, "targetfolder": true, "target": true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 1 && a[0] == '-' {
			name := strings.ToLower(strings.TrimLeft(a, "-"))
			value := ""
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				value = a[strings.IndexByte(a, '=')+1:]
				name = name[:eq]
			} else if takesValue[name] && i+1 < len(args) && !isFlag(args[i+1]) {
				i++
				value = args[i]
			}
			switch name {
			case "sourcefolder", "source":
				if value != "" {
					sources = append(sources, value)
				}
			case "targetfolder", "target":
				target = value
			case "edit":
				edit = true
			case "pick-template":
				pick = true
			}
			continue
		}
		if a != "" {
			sources = append(sources, a)
		}
	}

	in := Intent{Mode: ModeHome, Target: target}
	switch {
	case pick:
		in.Mode = ModePick
		// A folder passed to the pick menu is the destination, not a template.
		if in.Target == "" && len(sources) > 0 {
			in.Target = sources[0]
		}
		return in
	case len(sources) == 0:
		return in
	case edit:
		in.Mode = ModeEdit
	default:
		in.Mode = ModeOpen
	}
	in.Source = sources[0]
	in.Extra = sources[1:]
	return in
}

func isFlag(s string) bool { return len(s) > 1 && s[0] == '-' }

// ArgsFor rebuilds a command line that opens folder the same way intent
// opened its Source — used to give each extra SendTo selection its own window.
func ArgsFor(in Intent, folder string) []string {
	args := []string{}
	if in.Mode == ModeEdit {
		args = append(args, "-edit")
	}
	args = append(args, "-sourceFolder", folder)
	if in.Target != "" {
		args = append(args, "-targetFolder", in.Target)
	}
	return args
}
