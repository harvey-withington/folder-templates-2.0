# foldertemplate

Generate folder structures from templates. A Go port of [Folder Templates](https://github.com/harvey-withington/Folder-Templates) (C#, MIT) — same on-disk format, so templates authored by the original app, the Obsidian plugin, or by hand keep working.

Lives inside the BRUV monorepo today but is its own Go module with **zero BRUV imports** (compiler-enforced), declared at its eventual public path. Extraction later is a `git subtree split` — no import changes.

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

## API

```go
tpl, err := foldertemplate.Load(dir)               // reads .ft/template.json
issues  := tpl.Validate()                          // regex + name checks
entries, warns, err := foldertemplate.Preview(tpl, values, extra, nil) // dry run
res, err := foldertemplate.Generate(tpl, targetParent, values, extra, nil)
before, after, err := foldertemplate.RenderFile(tpl, "script.md.ft$", values, extra, nil)
err = foldertemplate.Save(tpl, dir)                // template editor writes
```

`values` answers declared parameters; `extra` supplies caller context (BRUV injects `bruvBrand`, `bruvStream`, `bruvProject`, `bruvDate`) resolvable without being declared — declared parameters with the same name win.

## Testing

```
go test ./...
```

`testdata/youtube-template` is authored in the C# app's PascalCase style and exercises the compatibility contract end-to-end.
