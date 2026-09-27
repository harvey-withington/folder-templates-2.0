package launch

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Intent
	}{
		{"no args", nil, Intent{Mode: ModeHome}},
		{"bare folder (1.0 Process SendTo)", []string{`C:\T\tpl`}, Intent{Mode: ModeOpen, Source: `C:\T\tpl`, Extra: []string{}}},
		{"1.0 Edit SendTo", []string{"-edit", "-sourceFolder", `C:\T\tpl`}, Intent{Mode: ModeEdit, Source: `C:\T\tpl`, Extra: []string{}}},
		{"1.0 restart form", []string{"-edit", "-sourceFolder", "a", "-targetFolder", "b"}, Intent{Mode: ModeEdit, Source: "a", Target: "b", Extra: []string{}}},
		{"multi-select SendTo", []string{"a", "b", "c"}, Intent{Mode: ModeOpen, Source: "a", Extra: []string{"b", "c"}}},
		{"edit shortcut with two folders", []string{"-edit", "-sourceFolder", "a", "b"}, Intent{Mode: ModeEdit, Source: "a", Extra: []string{"b"}}},
		{"v2 flags", []string{"--source", "a", "--target=b"}, Intent{Mode: ModeOpen, Source: "a", Target: "b", Extra: []string{}}},
		{"case-insensitive flags", []string{"-SOURCEFOLDER", "a"}, Intent{Mode: ModeOpen, Source: "a", Extra: []string{}}},
		{"pick with target", []string{"--pick-template", "--target", `C:\Here`}, Intent{Mode: ModePick, Target: `C:\Here`}},
		{"pick with bare folder is the target", []string{"--pick-template", `C:\Here`}, Intent{Mode: ModePick, Target: `C:\Here`}},
		{"unknown flags ignored", []string{"-nowait", "a"}, Intent{Mode: ModeOpen, Source: "a", Extra: []string{}}},
		{"value-less source flag", []string{"-sourceFolder"}, Intent{Mode: ModeHome}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Parse(c.args); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", c.args, got, c.want)
			}
		})
	}
}

func TestArgsForRoundTrips(t *testing.T) {
	for _, in := range []Intent{
		{Mode: ModeOpen, Source: "x"},
		{Mode: ModeEdit, Source: "x", Target: "t"},
	} {
		got := Parse(ArgsFor(in, "y"))
		if got.Mode != in.Mode || got.Source != "y" || got.Target != in.Target {
			t.Errorf("ArgsFor(%+v) re-parsed to %+v", in, got)
		}
	}
}
