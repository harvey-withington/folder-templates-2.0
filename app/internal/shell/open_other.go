//go:build !windows

package shell

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenFolder opens path in the file manager.
func OpenFolder(path string) error {
	return opener(path).Start()
}

// Reveal opens the folder containing path.
func Reveal(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}
	return opener(filepath.Dir(path)).Start()
}

func opener(path string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path)
	}
	return exec.Command("xdg-open", path)
}
