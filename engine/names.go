package foldertemplate

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrInvalidName is returned (wrapped) when a renamed output name can't be
// created — usually because a parameter value put a character in it that the
// filesystem forbids, such as "?" on Windows.
var ErrInvalidName = errors.New("invalid file or folder name")

// invalidNameError reads as one plain sentence and matches ErrInvalidName
// with errors.Is.
type invalidNameError struct{ name, reason string }

func (e *invalidNameError) Error() string {
	return fmt.Sprintf("%q can't be used as a file or folder name: %s", e.name, e.reason)
}

func (e *invalidNameError) Is(target error) bool { return target == ErrInvalidName }

// windowsReserved are device names Windows won't use as file or folder
// names, with or without an extension.
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM0": true, "COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT0": true, "LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// checkOutputName rejects a renamed name the OS can't create. On every OS it
// refuses path separators and "."/".." — values must never move output
// outside its folder. On Windows it also applies Windows' naming rules.
func checkOutputName(name string) error {
	return checkOutputNameFor(name, runtime.GOOS == "windows")
}

func checkOutputNameFor(name string, windows bool) error {
	bad := func(reason string) error { return &invalidNameError{name: name, reason: reason} }
	if name == "." || name == ".." {
		return bad("it refers to a folder, not a name")
	}
	if strings.ContainsRune(name, '/') || (windows && strings.ContainsRune(name, '\\')) {
		return bad("it contains a path separator")
	}
	if strings.ContainsRune(name, 0) {
		return bad("it contains a NUL character")
	}
	if !windows {
		return nil
	}
	for _, r := range name {
		if r < 32 {
			return bad("it contains a control character")
		}
		if strings.ContainsRune(`<>:"|?*`, r) {
			return bad(fmt.Sprintf("Windows doesn't allow %q in names", string(r)))
		}
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return bad("Windows doesn't allow names ending in a dot or space")
	}
	base := strings.ToUpper(name)
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	if windowsReserved[strings.TrimRight(base, " ")] {
		return bad("it is a reserved device name on Windows")
	}
	return nil
}
