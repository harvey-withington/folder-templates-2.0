package foldertemplate

import (
	"errors"
	"testing"
)

func TestCheckOutputName(t *testing.T) {
	cases := []struct {
		name    string
		windows bool
		ok      bool
	}{
		{"Episode 12", true, true},
		{"notes.md", true, true},
		{"Why is the sky?", true, false},
		{"Why is the sky?", false, true}, // legal on Linux/macOS
		{"a:b", true, false},
		{`a"b`, true, false},
		{"a|b", true, false},
		{"a*b", true, false},
		{"a<b>", true, false},
		{"trailing.", true, false},
		{"trailing ", true, false},
		{"CON", true, false},
		{"con.txt", true, false},
		{"LPT1", true, false},
		{"CONSOLE", true, true},
		{"tab\there", true, false},
		{"a/b", false, false},
		{"a/b", true, false},
		{`a\b`, true, false},
		{`a\b`, false, true},
		{"..", false, false},
		{".", true, false},
		{"..hidden", true, true},
		{"nul\x00", false, false},
	}
	for _, c := range cases {
		err := checkOutputNameFor(c.name, c.windows)
		if (err == nil) != c.ok {
			t.Errorf("checkOutputNameFor(%q, windows=%v) = %v, want ok=%v", c.name, c.windows, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrInvalidName) {
			t.Errorf("%q: error should wrap ErrInvalidName: %v", c.name, err)
		}
	}
}
