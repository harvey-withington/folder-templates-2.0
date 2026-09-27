# Parity corpus

Each case is a template plus the output Folder Templates 1.0's console
produces for it. `internal/cli/parity_test.go` runs every case through `ft`'s
1.0-compatible mode and compares the result byte for byte.

    <case>/case.json       parameter values (passed as -name value flags) and
                           emptyDirs to create in the template first (git
                           cannot store empty folders)
    <case>/template/<root> the template folder
    <case>/manifest.txt    every output path, folders ending in /
    <case>/expected/       expected file contents

The expected outputs were derived from the 1.0 source by hand. To confirm them
against the real 1.0 console, run `scripts/parity-regen.ps1` with a .NET 9 SDK
installed; it rewrites `expected/` and `manifest.txt` from what 1.0 actually
produces, and `git diff` shows any disagreement.

Cases steer clear of the documented deviations from 1.0 (see engine/README.md):
BOMs, dots in the template root name, mixed-case content tokens.
