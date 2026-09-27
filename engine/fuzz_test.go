package foldertemplate

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// FuzzProcessContent: content without a {{$ opener must pass through
// byte-for-byte, and nothing may panic.
func FuzzProcessContent(f *testing.F) {
	for _, s := range []string{"", "hello\n", "\xEF\xBB\xBFbom\r\nline", "{{$p}} and {{$P}}", "{{$", "no newline"} {
		f.Add(s)
	}
	b, err := newBinding(Parameter{Name: "p", ReplaceInFiles: true}, "VALUE", Options{MatchTimeout: time.Second})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var out bytes.Buffer
		err := processContent(strings.NewReader(in), &out, []binding{b})
		if strings.IndexByte(in, 0) >= 0 && len(in) <= binarySniffLen {
			if err != ErrBinaryContent {
				t.Fatalf("NUL input must be refused, got %v", err)
			}
			return
		}
		if err != nil {
			return
		}
		if !strings.Contains(strings.ToLower(in), "{{$p}}") && out.String() != in {
			t.Fatalf("token-free content changed: %q → %q", in, out.String())
		}
	})
}

// FuzzApplyToName: default-match renaming only ever touches the literal
// {name} token.
func FuzzApplyToName(f *testing.F) {
	for _, s := range []string{"{p}", "a{p}b{p}", "{P}", "plain.txt", "{{p}}", `\{p\}`} {
		f.Add(s, "v")
	}
	f.Fuzz(func(t *testing.T, name, value string) {
		b, err := newBinding(Parameter{Name: "p", ReplaceInFileNames: true}, value, Options{MatchTimeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		got, err := applyToName(name, []binding{b})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(name, "{p}") && got != name {
			t.Fatalf("%q renamed to %q without a {p} token", name, got)
		}
	})
}
