<#
.SYNOPSIS
  Builds app\build\bin\FolderTemplates-debug.exe for Delve: the Wails dev
  tag, optimizations off, and the app icon embedded.

.DESCRIPTION
  `wails build` embeds build\windows\icon.ico through a temporary .syso
  resource file; a plain `go build` doesn't, so the window would show a
  generic icon. This script makes an equivalent resource with go-winres
  (spec: build\windows\winres-debug.json),
  builds, and always deletes it again: left behind, it would clash with the
  one `wails build` generates.
#>
param(
    # Where to write the exe, relative to app\ (or absolute).
    [string]$Out = 'build/bin/FolderTemplates-debug.exe'
)
$ErrorActionPreference = 'Stop'
$app = Resolve-Path (Join-Path $PSScriptRoot '..\app')
$syso = Join-Path $app 'debug_windows_amd64.syso'

Push-Location $app
try {
    # Same resources `wails build` embeds: the icon as group-icon ID 3 (the ID
    # Wails loads for the window and taskbar) and Wails' manifest.
    go run github.com/tc-hib/go-winres@v0.3.3 make `
        --in build/windows/winres-debug.json --arch amd64 --out debug
    if ($LASTEXITCODE -ne 0) { throw "go-winres failed (exit $LASTEXITCODE)" }

    go build -tags dev -gcflags 'all=-N -l' -o $Out .
    if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }
}
finally {
    Remove-Item $syso -ErrorAction SilentlyContinue
    Pop-Location
}
