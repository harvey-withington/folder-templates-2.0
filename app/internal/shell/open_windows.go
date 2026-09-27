//go:build windows

package shell

import (
	"os/exec"
	"path/filepath"
	"syscall"
)

// OpenFolder opens path in Explorer.
func OpenFolder(path string) error {
	return explorer(`"` + filepath.Clean(path) + `"`)
}

// Reveal opens the parent folder in Explorer with path selected.
func Reveal(path string) error {
	return explorer(`/select,"` + filepath.Clean(path) + `"`)
}

// explorer starts explorer.exe with a raw command line: it parses its own
// arguments, and Go's quoting would break /select,"path with spaces".
// explorer.exe exits 1 even on success, so only a failure to start counts.
func explorer(args string) error {
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "explorer.exe " + args}
	return cmd.Start()
}
