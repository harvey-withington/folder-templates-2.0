# foldertemplate

Generate folder structures from templates. A Go port of [Folder Templates](https://github.com/harvey-withington/Folder-Templates) (C#, MIT) — same on-disk format, so templates authored by the original app, the Obsidian plugin, or by hand keep working.

Home: the Folder Templates v2 repo (`engine/`), which also builds the desktop app and the `ft` CLI on it. Originally written inside BRUV (copied from `bruv-1.0/foldertemplate` @ 8f38eb6); BRUV will consume tagged releases of this module.

## Template format

A template is an ordinary folder representing the desired structure, plus a `.ft/template.json` descriptor. The `.ft/` directory is never copied to output.

```json
{
  "name": "YouTube Video Project",
  "description": "Standard video structure",
  "defaultTargetPath": "D:/Projects/Videos",
  "parameters": [
    {
      "name": "videoName",
      "type": "text",
      "prompt": "Video name?",
      "placeholder": "e.g. episode-042",
      "defaultValue": null,
      "match": "\\{videoName\\}",
      "replaceInFileNames": true,
      "replaceInFiles": true
    }
  ]
}
```

Keys are read case-insensitively (the C# app historically wrote PascalCase) and written camelCase.

### Semantics (preserved from the original)

- **Name replacement** — parameters with `replaceInFileNames` regex-replace (`match` → value) every file/folder name during copy, including the template's root folder name. Default `match`: the literal `\{name\}`.
- **Content replacement** — only files with the extra extension `.ft$` are processed: `{{$param}}` tokens (case-insensitive) are replaced for parameters with `replaceInFiles`; unknown tokens pass through; the `.ft$` suffix is stripped. Everything else is copied byte-for-byte.
- **Visibility** — parameters without a `prompt` are internal (filled from `defaultValue` or caller context).
- Missing values resolve to `value ?? defaultValue ?? ""`.
- `match` patterns compile with [regexp2](https://github.com/dlclark/regexp2) for full .NET regex parity (backreferences, lookaround). A `MatchTimeout` (default 2 s) guards against catastrophic backtracking in untrusted templates.

### Fixes over the C# original

- Recursion guard: generating into the template folder itself is refused.
- `.ft$` files have a size ceiling (default 10 MB) and a binary sniff (NUL bytes → clear error instead of corruption); UTF-8 BOMs are preserved.
- Case-only output collisions (post-rename) are detected before anything is written.
- Symlinks are never followed; they're skipped with a warning.
- An existing output folder is refused by default instead of silently merged into. `Options.Conflict` restores the original's behaviour on request (`ConflictOverwrite`) or merges without replacing files (`ConflictMerge`). Overwrites are atomic per file.

### Known differences from the C# original

Deliberate, and covered by tests:

- Content tokens match parameter names case-insensitively (`{{$VideoName}}` fills `videoName`); the original compared the name case-sensitively.
- The output root keeps the template folder's full name; the original dropped anything after the last dot (`Path.GetFileNameWithoutExtension`), so `v1.2 {name}` became `v1`.
- The `.ft$` suffix is matched case-sensitively on the renamed name; the original matched the source extension case-insensitively (`.FT$`).
- An empty `match` string means "use the default"; in the original it matched between every character.

## API

```go
tpl, err := foldertemplate.Load(dir)               // reads .ft/template.json
issues  := tpl.Validate()                          // regex + name checks
entries, warns, err := foldertemplate.Preview(tpl, values, extra, nil) // dry run
res, err := foldertemplate.Generate(tpl, targetParent, values, extra, nil)
before, after, err := foldertemplate.RenderFile(tpl, "script.md.ft$", values, extra, nil)
err = foldertemplate.Save(tpl, dir)                // template editor writes (.ft hidden on Windows)
```

Additions for editors and front-ends:

```go
res, err := foldertemplate.GenerateContext(ctx, tpl, target, values, extra,
    &foldertemplate.Options{Conflict: foldertemplate.ConflictMerge, Progress: onProgress})
pr, err  := foldertemplate.PreviewAt(tpl, target, values, extra, nil)   // + exists flags, recursion guard
dst      := foldertemplate.ResolveDefaultTarget(tpl)                    // the original's default-target rule
insp, err := foldertemplate.Inspect(dir)                                // template or plain folder, size, issues
tree, err := foldertemplate.Tree(dir)                                   // source tree, .ft/ excluded
scan, err := foldertemplate.Scan(dir, unsavedTpl, nil)                  // tokens, param usage, authoring issues
hits, err := foldertemplate.TestMatch(dir, `^_Template - `, "", nil)    // what a match pattern would rename
newRel, err := foldertemplate.SetContentProcessing(dir, "notes.md", true) // notes.md → notes.md.ft$
```

`values` answers declared parameters; `extra` supplies caller context (BRUV injects `bruvBrand`, `bruvStream`, `bruvProject`, `bruvDate`) resolvable without being declared — declared parameters with the same name win.

## Testing

```
go test ./...
go test -run=^$ -fuzz=FuzzProcessContent -fuzztime=30s .
go test -run=^$ -fuzz=FuzzApplyToName -fuzztime=30s .
```

`testdata/youtube-template` is authored in the C# app's PascalCase style and exercises the compatibility contract end-to-end.
