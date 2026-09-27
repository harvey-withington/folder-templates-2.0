package foldertemplate

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// Options tunes generation. The zero value / nil means defaults.
type Options struct {
	// MatchTimeout bounds each regexp2 (backtracking, no linear-time
	// guarantee) name-replacement evaluation. Default 2s.
	MatchTimeout time.Duration
	// ContentSizeLimit is the max size of a .ft$ file. Default 10 MB.
	ContentSizeLimit int64
}

func (o *Options) withDefaults() Options {
	out := Options{MatchTimeout: 2 * time.Second, ContentSizeLimit: 10 << 20}
	if o != nil {
		if o.MatchTimeout > 0 {
			out.MatchTimeout = o.MatchTimeout
		}
		if o.ContentSizeLimit > 0 {
			out.ContentSizeLimit = o.ContentSizeLimit
		}
	}
	return out
}

// effectiveMatch returns the parameter's name-replacement pattern:
// the declared Match, or the original's default — the literal `\{Name\}`.
func effectiveMatch(p Parameter) string {
	if p.Match != nil && *p.Match != "" {
		return *p.Match
	}
	return `\{` + regexp.QuoteMeta(p.Name) + `\}`
}

// binding is a parameter resolved against caller values: Value ?? DefaultValue ?? "".
type binding struct {
	param Parameter
	value string

	nameRe    *regexp2.Regexp // compiled Match; nil unless ReplaceInFileNames
	contentRe *regexp.Regexp  // (?i)\{\{\$name\}\}; nil unless ReplaceInFiles
}

// resolveBindings merges declared parameters with caller-supplied extras.
// Declared parameters win on name conflicts (case-insensitive) — extras are
// context values (e.g. BRUV's bruvProject/bruvDate) that behave as if declared
// with default Match and both replace flags on. Order: declared first, in
// declaration order, then extras (deterministic by insertion? no — sorted by
// the caller if order matters; extras are order-independent because their
// default matches are disjoint literals).
func resolveBindings(t *Template, values, extra map[string]string, opts Options) ([]binding, error) {
	lookup := func(m map[string]string, name string) (string, bool) {
		if v, ok := m[name]; ok {
			return v, true
		}
		for k, v := range m {
			if strings.EqualFold(k, name) {
				return v, true
			}
		}
		return "", false
	}

	declared := map[string]bool{}
	var out []binding
	for _, p := range t.Parameters {
		declared[strings.ToLower(p.Name)] = true
		v, ok := lookup(values, p.Name)
		if !ok && p.DefaultValue != nil {
			v = *p.DefaultValue
		}
		b, err := newBinding(p, v, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	for name, v := range extra {
		if declared[strings.ToLower(name)] {
			continue // declared parameters win, for compatibility
		}
		p := Parameter{Name: name, Type: "text", ReplaceInFileNames: true, ReplaceInFiles: true}
		b, err := newBinding(p, v, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func newBinding(p Parameter, value string, opts Options) (binding, error) {
	b := binding{param: p, value: value}
	if p.ReplaceInFileNames {
		re, err := regexp2.Compile(effectiveMatch(p), regexp2.None)
		if err != nil {
			return b, &ValidationError{Param: p.Name, Err: fmt.Errorf("invalid match pattern: %w", err)}
		}
		re.MatchTimeout = opts.MatchTimeout
		b.nameRe = re
	}
	if p.ReplaceInFiles {
		// Content tokens are fixed-form {{$name}}, case-insensitive — stdlib
		// regexp suffices (the pattern is ours, not the template author's).
		b.contentRe = regexp.MustCompile(`(?i)\{\{\$` + regexp.QuoteMeta(p.Name) + `\}\}`)
	}
	return b, nil
}

// applyToName runs every name replacement over one path segment, sequentially
// in binding order (matching the original's per-parameter replace loop).
func applyToName(name string, bindings []binding) (string, error) {
	for _, b := range bindings {
		if b.nameRe == nil {
			continue
		}
		replaced, err := b.nameRe.Replace(name, b.value, -1, -1)
		if err != nil {
			// regexp2 returns an error on MatchTimeout — surface as validation.
			return "", &ValidationError{Param: b.param.Name, Err: fmt.Errorf("match timed out or failed on %q: %w", name, err)}
		}
		name = replaced
	}
	return name, nil
}

// applyToContent replaces {{$param}} tokens on one line. Unknown tokens pass
// through untouched by construction (only known parameters have patterns).
func applyToContent(line string, bindings []binding) string {
	for _, b := range bindings {
		if b.contentRe == nil {
			continue
		}
		line = b.contentRe.ReplaceAllLiteralString(line, b.value)
	}
	return line
}

// ValidationError marks template-authoring problems (bad or pathological
// Match patterns) as distinct from I/O failures — the UI shows these on the
// template, not as a generation crash.
type ValidationError struct {
	Param string
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("parameter %q: %v", e.Param, e.Err)
}

func (e *ValidationError) Unwrap() error { return e.Err }

// Validate checks the template descriptor: named parameters unique
// (case-insensitive), Match patterns compile under regexp2. Returns one error
// per problem.
//
// Anonymous parameters (empty Name) are legal when they declare a Match —
// the C# app uses them as pure rename/deletion rules (e.g. stripping a
// "^_Template - " prefix from the root folder name). They just can't carry a
// content token, so an empty name without a Match is useless and flagged.
func (t *Template) Validate() []error {
	var errs []error
	seen := map[string]bool{}
	for _, p := range t.Parameters {
		if p.Name == "" {
			if p.Match == nil || *p.Match == "" {
				errs = append(errs, &ValidationError{Param: "", Err: fmt.Errorf("anonymous parameter needs a match pattern")})
				continue
			}
		} else {
			lower := strings.ToLower(p.Name)
			if seen[lower] {
				errs = append(errs, &ValidationError{Param: p.Name, Err: fmt.Errorf("duplicate parameter name")})
			}
			seen[lower] = true
		}
		if _, err := regexp2.Compile(effectiveMatch(p), regexp2.None); err != nil {
			errs = append(errs, &ValidationError{Param: p.Name, Err: fmt.Errorf("invalid match pattern: %w", err)})
		}
	}
	return errs
}
