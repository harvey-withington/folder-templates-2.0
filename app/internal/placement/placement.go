// Package placement opens the main window centred on the display the app
// was launched from: the one showing the Explorer window whose context menu
// started it, or the one under the mouse.
//
// Only the window's size is remembered between runs, never its position.
// Wails' WindowSetPosition is relative to the monitor the window is on while
// WindowGetPosition is absolute, so a saved position lands off-screen or on
// the wrong display, and any saved position goes stale when monitors are
// plugged, unplugged or rearranged.
package placement

// Rect is a rectangle in physical screen pixels; Right and Bottom are
// exclusive, as in Win32's RECT.
type Rect struct {
	Left, Top, Right, Bottom int
}

// Width of r.
func (r Rect) Width() int { return r.Right - r.Left }

// Height of r.
func (r Rect) Height() int { return r.Bottom - r.Top }

// Centre returns a width × height rectangle centred in work, shrunk to fit
// when it is larger.
func Centre(work Rect, width, height int) Rect {
	width = min(width, work.Width())
	height = min(height, work.Height())
	left := work.Left + (work.Width()-width)/2
	top := work.Top + (work.Height()-height)/2
	return Rect{Left: left, Top: top, Right: left + width, Bottom: top + height}
}

// scale converts a size in device-independent pixels (what Wails and the
// settings use) to physical pixels at dpi.
func scale(dips, dpi int) int {
	return (dips*dpi + 48) / 96
}
