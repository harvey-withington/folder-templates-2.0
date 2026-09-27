<#
.SYNOPSIS
  Regenerates testdata/parity expected outputs with the real Folder Templates 1.0 console.

.DESCRIPTION
  Builds FolderTemplates.Console from the 1.0 source (needs a .NET 9 SDK), runs
  every parity case through it with the same flags the Go parity test uses, and
  rewrites each case's expected/ folder and manifest.txt. Review with git diff:
  any change is a place where ft and 1.0 disagree.

.PARAMETER V1Source
  Path to the folder-templates-1.0 checkout. Defaults to the sibling folder.
#>
param(
    [string]$V1Source = (Join-Path $PSScriptRoot '..\..\folder-templates-1.0')
)
$ErrorActionPreference = 'Stop'

$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$proj = Join-Path $V1Source 'FolderTemplates.Console\FolderTemplates.Console.csproj'
if (-not (Test-Path $proj)) { throw "1.0 console project not found at $proj" }

$bin = Join-Path ([IO.Path]::GetTempPath()) 'ft-parity-v1'
dotnet build $proj -c Release -o $bin | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'dotnet build failed' }
$exe = Join-Path $bin 'FolderTemplates.Console.exe'

Get-ChildItem (Join-Path $repo 'testdata\parity') -Directory | ForEach-Object {
    $case = $_.FullName
    $spec = Get-Content (Join-Path $case 'case.json') -Raw | ConvertFrom-Json

    $work = Join-Path ([IO.Path]::GetTempPath()) ("ft-parity-" + $_.Name)
    if (Test-Path -LiteralPath $work) { Remove-Item -LiteralPath $work -Recurse -Force }
    New-Item -ItemType Directory -Path (Join-Path $work 'src'), (Join-Path $work 'out') | Out-Null

    $root = Get-ChildItem -LiteralPath (Join-Path $case 'template') -Directory | Select-Object -First 1
    Copy-Item -LiteralPath $root.FullName -Destination (Join-Path $work 'src') -Recurse
    $src = Join-Path (Join-Path $work 'src') $root.Name
    foreach ($d in @($spec.emptyDirs)) {
        if ($d) { New-Item -ItemType Directory -Force -Path (Join-Path $src $d) | Out-Null }
    }

    $argv = @('-sourceFolder', $src, '-targetFolder', (Join-Path $work 'out'), '-noprompt', '-nowait')
    foreach ($p in ($spec.values.PSObject.Properties | Sort-Object Name)) { $argv += @("-$($p.Name)", $p.Value) }
    & $exe @argv | Out-Host

    $out = Join-Path $work 'out'
    $expected = Join-Path $case 'expected'
    if (Test-Path -LiteralPath $expected) { Remove-Item -LiteralPath $expected -Recurse -Force }
    Copy-Item -LiteralPath $out -Destination $expected -Recurse

    $manifest = Get-ChildItem -LiteralPath $out -Recurse -Force | ForEach-Object {
        $rel = $_.FullName.Substring($out.Length + 1).Replace('\', '/')
        if ($_.PSIsContainer) { "$rel/" } else { $rel }
    } | Sort-Object -CaseSensitive
    [IO.File]::WriteAllText((Join-Path $case 'manifest.txt'), (($manifest -join "`n") + "`n"))
    Write-Host "regenerated $($_.Name)"
}
