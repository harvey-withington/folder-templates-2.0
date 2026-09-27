package foldertemplate

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
)

// Scan issue kinds.
const (
	// IssueTokenInUnprocessedFile: a {{$token}} sits in a file not named
	// *.ft$, so it will be copied verbatim — the most common authoring slip.
	IssueTokenInUnprocessedFile = "token-in-unprocessed-file"
	// IssueUndeclaredToken: a token appears in names or content but no
	// parameter declares it.
	IssueUndeclaredToken = "undeclared-token"
	// IssueUnusedParameter: a parameter matches no name and no content.
	IssueUnusedParameter = "unused-parameter"
	// IssueNameReplacementOff: a {token} in a name matches a parameter whose
	// replaceInFileNames is off.
	IssueNameReplacementOff = "name-replacement-off"
	// IssueContentReplacementOff: a {{$token}} in a .ft$ file matches a
	// parameter whose replaceInFiles is off.
	IssueContentReplacementOff = "content-replacement-off"
	// IssueUnprocessableContent: a .ft$ file is binary or over the size limit.
	IssueUnprocessableContent = "unprocessable-content-file"
)

// ScanResult is what Scan found in a template folder.
type ScanResult struct {
	Tokens []TokenInfo  `json:"tokens"`
	Params []ParamUsage `json:"params"`
	Issues []ScanIssue  `json:"issues"`
}

// TokenInfo is one distinct token (case-insensitive) and where it appears.
// Paths are slash-separated and relative to the template folder; "." is the
// template folder's own name.
type TokenInfo struct {
	Name      string   `json:"name"`
	InNames   []string `json:"inNames"`   // {token} in file/folder names
	InContent []string `json:"inContent"` // {{$token}} in .ft$ files
	Declared  bool     `json:"declared"`
}

// ParamUsage says how often a declared parameter would actually fire.
type ParamUsage struct {
	Name        string `json:"name"`
	Match       string `json:"match"`       // effective name pattern
	NameHits    int    `json:"nameHits"`    // names its match renames (if replaceInFileNames)
	ContentHits int    `json:"contentHits"` // .ft$ files containing its token (if replaceInFiles)
}

// ScanIssue is one authoring problem.
type ScanIssue struct {
	Kind    string `json:"kind"`
	Param   string `json:"param,omitempty"`
	Token   string `json:"token,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

var (
	nameTokenRe    = regexp.MustCompile(`\{(\w+)\}`)
	contentTokenRe = regexp.MustCompile(`(?i)\{\{\$(\w+)\}\}`)
)

// unprocessedSniffLimit caps how much of a non-.ft$ file Scan reads when
// looking for misplaced {{$tokens}}.
const unprocessedSniffLimit = 1 << 20

// Scan inspects a template folder for tokens and authoring mistakes. tpl is
// the descriptor to check against — pass the editor's unsaved copy, or nil to
// load dir's own .ft/template.json (a folder without one scans as having no
// parameters, which is how "make this a template" gets its suggestions).
func Scan(dir string, tpl *Template, opts *Options) (*ScanResult, error) {
	o := opts.withDefaults()
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		loaded, err := Load(abs)
		switch {
		case err == nil:
			tpl = loaded
		case errors.Is(err, ErrNotATemplate):
			tpl = &Template{}
		default:
			return nil, err
		}
	}

	entries, err := Tree(abs)
	if err != nil {
		return nil, err
	}

	res := &ScanResult{Tokens: []TokenInfo{}, Params: []ParamUsage{}, Issues: []ScanIssue{}}
	tokens := map[string]*TokenInfo{}
	var order []string
	token := func(name string) *TokenInfo {
		key := strings.ToLower(name)
		if ti, ok := tokens[key]; ok {
			return ti
		}
		ti := &TokenInfo{Name: name, InNames: []string{}, InContent: []string{}}
		tokens[key] = ti
		order = append(order, key)
		return ti
	}

	// names — the root folder's name included, reported as "."
	type named struct{ rel, name string }
	names := []named{{".", filepath.Base(abs)}}
	for _, e := range entries {
		names = append(names, named{e.Rel, path.Base(e.Rel)})
	}
	for _, n := range names {
		for _, m := range nameTokenRe.FindAllStringSubmatch(n.name, -1) {
			ti := token(m[1])
			ti.InNames = appendOnce(ti.InNames, n.rel)
		}
	}

	// contents
	processedText := map[string]string{} // rel → content of readable .ft$ files
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		full := filepath.Join(abs, filepath.FromSlash(e.Rel))
		if e.Processed {
			if e.Size > o.ContentSizeLimit {
				res.Issues = append(res.Issues, ScanIssue{Kind: IssueUnprocessableContent, Path: e.Rel,
					Message: fmt.Sprintf("%s is larger than the %d-byte content limit and will fail to generate", e.Rel, o.ContentSizeLimit)})
				continue
			}
			raw, err := os.ReadFile(full)
			if err != nil {
				return nil, err
			}
			if looksBinary(raw) {
				res.Issues = append(res.Issues, ScanIssue{Kind: IssueUnprocessableContent, Path: e.Rel,
					Message: fmt.Sprintf("%s looks binary; only text files can use %s", e.Rel, ContentExt)})
				continue
			}
			text := string(raw)
			processedText[e.Rel] = text
			for _, m := range contentTokenRe.FindAllStringSubmatch(text, -1) {
				ti := token(m[1])
				ti.InContent = appendOnce(ti.InContent, e.Rel)
			}
			continue
		}
		if e.Size > unprocessedSniffLimit {
			continue
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		if looksBinary(raw) {
			continue
		}
		var found []string
		for _, m := range contentTokenRe.FindAllStringSubmatch(string(raw), -1) {
			found = appendOnce(found, m[0])
		}
		if len(found) > 0 {
			res.Issues = append(res.Issues, ScanIssue{Kind: IssueTokenInUnprocessedFile, Path: e.Rel, Token: strings.Join(found, ", "),
				Message: fmt.Sprintf("%s contains %s but is not named *%s, so it is copied without replacement", e.Rel, strings.Join(found, ", "), ContentExt)})
		}
	}

	// parameters
	params := map[string]Parameter{}
	for _, p := range tpl.Parameters {
		if p.Name != "" {
			params[strings.ToLower(p.Name)] = p
		}
	}
	for _, p := range tpl.Parameters {
		u := ParamUsage{Name: p.Name, Match: effectiveMatch(p)}
		if p.ReplaceInFileNames {
			re, err := regexp2.Compile(u.Match, regexp2.None)
			if err == nil {
				re.MatchTimeout = o.MatchTimeout
				for _, n := range names {
					if ok, err := re.MatchString(n.name); err == nil && ok {
						u.NameHits++
					}
				}
			}
		}
		if p.ReplaceInFiles && p.Name != "" {
			tokRe := regexp.MustCompile(`(?i)\{\{\$` + regexp.QuoteMeta(p.Name) + `\}\}`)
			for _, text := range processedText {
				if tokRe.MatchString(text) {
					u.ContentHits++
				}
			}
		}
		res.Params = append(res.Params, u)
		if u.NameHits == 0 && u.ContentHits == 0 {
			label := p.Name
			if label == "" {
				label = u.Match
			}
			res.Issues = append(res.Issues, ScanIssue{Kind: IssueUnusedParameter, Param: p.Name,
				Message: fmt.Sprintf("parameter %s matches no file or folder name and no %s content", label, ContentExt)})
		}
	}

	// tokens vs parameters
	sort.Strings(order)
	for _, key := range order {
		ti := tokens[key]
		p, declared := params[key]
		ti.Declared = declared
		switch {
		case !declared:
			res.Issues = append(res.Issues, ScanIssue{Kind: IssueUndeclaredToken, Token: ti.Name,
				Message: fmt.Sprintf("token %s is used but no parameter declares it", ti.Name)})
		default:
			if len(ti.InNames) > 0 && !p.ReplaceInFileNames {
				res.Issues = append(res.Issues, ScanIssue{Kind: IssueNameReplacementOff, Param: p.Name, Token: ti.Name,
					Message: fmt.Sprintf("{%s} appears in names but parameter %s has name replacement off", ti.Name, p.Name)})
			}
			if len(ti.InContent) > 0 && !p.ReplaceInFiles {
				res.Issues = append(res.Issues, ScanIssue{Kind: IssueContentReplacementOff, Param: p.Name, Token: ti.Name,
					Message: fmt.Sprintf("{{$%s}} appears in %s files but parameter %s has content replacement off", ti.Name, ContentExt, p.Name)})
			}
		}
		res.Tokens = append(res.Tokens, *ti)
	}
	return res, nil
}

// NameHit is one name a pattern would rename.
type NameHit struct {
	SourceRel string `json:"sourceRel"` // "." is the template folder itself
	Name      string `json:"name"`
	Result    string `json:"result"`
	IsDir     bool   `json:"isDir"`
}

// TestMatch runs one name pattern (with the C# app's .NET regex semantics and
// the usual timeout) over every file and folder name in dir, the template
// folder's own name included, and returns the names it would rename. It
// works on the folder as it is on disk, independent of any saved descriptor.
func TestMatch(dir, pattern, replacement string, opts *Options) ([]NameHit, error) {
	o := opts.withDefaults()
	if pattern == "" {
		return nil, errors.New("empty match pattern")
	}
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("invalid match pattern: %w", err)
	}
	re.MatchTimeout = o.MatchTimeout
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	entries, err := Tree(abs)
	if err != nil {
		return nil, err
	}
	all := append([]TreeEntry{{Rel: ".", IsDir: true}}, entries...)
	hits := []NameHit{}
	for _, e := range all {
		name := path.Base(e.Rel)
		if e.Rel == "." {
			name = filepath.Base(abs)
		}
		ok, err := re.MatchString(name)
		if err != nil {
			return nil, fmt.Errorf("match timed out or failed on %q: %w", name, err)
		}
		if !ok {
			continue
		}
		out, err := re.Replace(name, replacement, -1, -1)
		if err != nil {
			return nil, fmt.Errorf("match timed out or failed on %q: %w", name, err)
		}
		hits = append(hits, NameHit{SourceRel: e.Rel, Name: name, Result: out, IsDir: e.IsDir})
	}
	return hits, nil
}

func looksBinary(raw []byte) bool {
	head := raw
	if len(head) > binarySniffLen {
		head = head[:binarySniffLen]
	}
	return bytes.IndexByte(head, 0) >= 0
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}
