//go:build !windows

package placement

import "errors"

// WindowClass is unused off Windows.
const WindowClass = ""

// Monitor is the display the window opens on.
type Monitor uintptr

// LaunchMonitor is unknown off Windows.
func LaunchMonitor() Monitor { return 0 }

// Place is unsupported off Windows; callers fall back to Wails' centring.
func Place(Monitor, int, int) error {
	return errors.New("window placement is only implemented on Windows")
}
