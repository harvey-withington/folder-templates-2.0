<#
.SYNOPSIS
  Tests and builds Folder Templates: the desktop app, ft.exe, the portable zip
  and (with -Installer) the NSIS installer.

.EXAMPLE
  ./scripts/build.ps1                     # test + build dev version
  ./scripts/build.ps1 -Version 2.0.0 -Installer
  ./scripts/build.ps1 -SkipTests          # quick rebuild
#>
param(
    [string]$Version = '2.0.0-dev',
    [switch]$Installer,
    [switch]$SkipTests
)
$ErrorActionPreference = 'Stop'
$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$dist = Join-Path $repo 'dist'
$bin = Join-Path $repo 'app\build\bin'

function Step($name, [scriptblock]$body) {
    Write-Host "`n== $name" -ForegroundColor Cyan
    # Native tools (wails, npm) write progress to stderr; Windows PowerShell
    # would turn that into a terminating error under 'Stop'. Judge them by
    # exit code instead.
    $global:LASTEXITCODE = 0
    $saved = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try { & $body } finally { $ErrorActionPreference = $saved }
    if ($LASTEXITCODE -ne 0) { throw "$name failed (exit $LASTEXITCODE)" }
}

Push-Location $repo
try {
    Step 'npm install' { npm install --no-audit --no-fund }

    if (-not $SkipTests) {
        Step 'engine tests' { Push-Location engine; go test ./...; Pop-Location }
        Step 'cli tests' { go test ./... }
        Step 'app Go tests' { Push-Location app; go test ./internal/...; Pop-Location }
        Step 'svelte-check' { npm run check }
        Step 'frontend tests' { npm test }
    }

    $ldflags = "-X main.Version=$Version"
    # ft.exe first: the NSIS step inside `wails build -nsis` packages it, and
    # `wails build -clean` empties build\bin, so it is built beside that.
    $ftExe = Join-Path $repo 'app\build\ft\ft.exe'
    Step 'ft.exe' { go build -trimpath -ldflags $ldflags -o $ftExe ./cmd/ft }
    Step 'wails build' {
        Push-Location app
        $wailsArgs = @('build', '-clean', '-platform', 'windows/amd64', '-trimpath', '-ldflags', $ldflags)
        if ($Installer) { $wailsArgs += '-nsis' }
        wails @wailsArgs
        Pop-Location
    }
    Copy-Item $ftExe $bin
    Step 'go vet (app, with the built frontend)' { Push-Location app; go vet ./...; Pop-Location }

    Step 'portable zip' {
        New-Item -ItemType Directory -Force -Path $dist | Out-Null
        $stage = Join-Path $dist 'stage'
        if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
        New-Item -ItemType Directory -Path $stage | Out-Null
        Copy-Item (Join-Path $bin 'FolderTemplates.exe'), (Join-Path $bin 'ft.exe') $stage
        Copy-Item (Join-Path $repo 'samples') (Join-Path $stage 'samples') -Recurse
        Copy-Item (Join-Path $repo 'LICENSE') $stage
        Set-Content -Path (Join-Path $stage 'portable') -Value 'Settings are kept next to FolderTemplates.exe while this file exists.'
        $zip = Join-Path $dist "FolderTemplates-$Version-win-x64-portable.zip"
        if (Test-Path $zip) { Remove-Item $zip }
        Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip
        Remove-Item $stage -Recurse -Force
    }

    if ($Installer) {
        Step 'collect installer' {
            $setup = Get-ChildItem $bin -Filter '*installer*.exe' | Select-Object -First 1
            if (-not $setup) { throw 'NSIS installer not found (is makensis on PATH?)' }
            Copy-Item $setup.FullName (Join-Path $dist "FolderTemplates-$Version-win-x64-setup.exe")
        }
    }

    Step 'checksums' {
        Get-ChildItem $dist -File -Include *.zip, *.exe -Recurse | ForEach-Object {
            "$((Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower())  $($_.Name)"
        } | Set-Content (Join-Path $dist 'SHA256SUMS.txt')
    }
    Write-Host "`nDone: $dist" -ForegroundColor Green
}
finally {
    Pop-Location
}
