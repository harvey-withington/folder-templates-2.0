# Folder Templates

Create folder structures from templates. Point Folder Templates at a template
folder, answer a few questions, and it copies the folder wherever you want,
with names and file contents filled in.

Version 2 is a rewrite of the [original WinForms app](https://github.com/HPoppington/Folder-Templates)
in Go, [Wails](https://wails.io) and Svelte. Templates made for 1.0 work
unchanged, and so do 1.0's SendTo shortcuts and command lines.

- **Desktop app** (`FolderTemplates.exe`) with a template library, live
  preview of what will be created, and a template editor.
- **Command line** (`ft.exe`) for scripts, including 1.0's console syntax.
- **Explorer integration**: Send to, plus right-click menus, including
  *New from template here…* inside any folder.

## Install

Download the latest release and either:

- run `FolderTemplates-<version>-win-x64-setup.exe` (installs for your account
  only; no administrator rights needed), or
- unzip `FolderTemplates-<version>-win-x64-portable.zip` anywhere. The portable
  build keeps its settings next to the exe.

The builds aren't code-signed, so Windows SmartScreen may say it "protected
your PC" the first time. Click **More info → Run anyway**.

To add Explorer shortcuts, open **Settings → Windows Explorer** and tick the
ones you want. The installer offers the same options.

## Make a template

A template is an ordinary folder laid out the way you want new folders to
look, plus a hidden `.ft` folder holding `template.json`. The easiest way to
make one is to drop a folder onto the app, or send it to
**Folder Template - Edit**. The editor creates `.ft/template.json` for you.

### Parameters fill in names

Each parameter is something the app asks for when you generate. Write
`{name}` in any file or folder name, including the template folder's own
name, and it is replaced with the answer:

```
{client} - {project}/            →  Acme - Relaunch/
  {project} brief.md             →    Relaunch brief.md
```

A parameter's **match** can be any .NET regular expression instead of the
default `{name}`. A parameter with no name and just a match is a rename rule.
For example, the match `^_Template - ` with an empty value strips that prefix
from every name.

### `.ft$` files get their contents filled in

Only files whose names end in `.ft$` have their contents processed. Inside
them, `{{$name}}` is replaced with the answer, and the `.ft$` is dropped from
the name:

```
README.md.ft$   containing   # {{$project}} for {{$client}}
→ README.md     containing   # Relaunch for Acme
```

Every other file is copied byte for byte. Tokens that no parameter declares
are left as they are. The editor's **Scan** panel finds tokens you haven't
declared yet, and flags `{{$tokens}}` sitting in files that aren't named `.ft$`.

### Other parameter settings

| Setting | Meaning |
|---|---|
| Prompt | The question shown. A parameter with no prompt is *internal*: it is never asked and always uses its default. |
| Placeholder | Hint text in the empty field. |
| Default | Used when the answer is left blank. |
| Replace in names / files | Where the parameter applies. |

`defaultTargetPath` in `template.json` is where new folders go by default. A
relative path is resolved against the folder that contains the template.

See `samples/Kitchen Sink` for every feature in one template.

### template.json

```json
{
  "name": "Video Episode",
  "description": "Script, footage and audio folders for one episode.",
  "defaultTargetPath": "../Episodes",
  "parameters": [
    {
      "name": "title",
      "type": "text",
      "prompt": "Episode title",
      "placeholder": "e.g. Why is the sky?",
      "defaultValue": null,
      "match": null,
      "replaceInFileNames": true,
      "replaceInFiles": true
    }
  ]
}
```

Keys are read case-insensitively, so 1.0's PascalCase files load too, and are
written in camelCase.

## Command line

```
ft generate <template> [--target DIR] [--set name=value]... [--on-conflict refuse|merge|overwrite] [--dry-run] [--json]
ft params   <template> [--all] [--json]
ft info     <template> [--json]
ft validate <template> [--json]
ft scan     <template> [--json]
```

If the output folder already exists, `ft` stops (exit code 3) unless you pass
`--on-conflict merge` (add what's missing) or `--on-conflict overwrite`
(replace existing files).

1.0's console syntax still works, for example
`ft -sourceFolder <template> -targetFolder <dir> -client Acme -nowait -noprompt`.
In that mode, `-listParams json` and `-getTemplateInfo json` produce the same
JSON as 1.0.

## Differences from 1.0

- An existing output folder is no longer silently merged into; you choose
  what happens.
- `.ft$` files keep a UTF-8 byte-order mark if they have one.
- `{{$Token}}` matches parameter names regardless of case.
- A template folder name containing a dot keeps its full name (1.0 cut it at
  the last dot).
- Generating into the template folder itself is refused instead of looping
  forever.

## Building

Requirements: Go 1.26, Node 24, the [Wails CLI](https://wails.io) v2.10, and
[NSIS](https://nsis.sourceforge.io) for the installer.

```powershell
./scripts/build.ps1                         # tests, app, ft.exe, portable zip
./scripts/build.ps1 -Version 2.0.0 -Installer
cd app; wails dev                           # run the app with hot reload
```

| Folder | What |
|---|---|
| `engine/` | The template engine, its own Go module (`github.com/harvey-withington/foldertemplate`), also used by BRUV |
| `cmd/ft`, `internal/cli` | The `ft` command line |
| `app/` | The Wails desktop app |
| `packages/ui/` | Shared Svelte components (`@harvey-withington/folder-templates-ui`) |
| `samples/` | Sample templates shipped with the app |
| `testdata/parity/` | Output compared byte for byte against 1.0 |

## Licence

MIT
