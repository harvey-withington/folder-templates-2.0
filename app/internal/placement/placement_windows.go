//go:build windows

package placement

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WindowClass is the main window's Win32 class name, set through Wails'
// windows.Options so Place can find the window.
const WindowClass = "FolderTemplatesWindow"

// Monitor is the display the window opens on.
type Monitor uintptr

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	shcore = windows.NewLazySystemDLL("shcore.dll")

	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procMonitorFromWindow   = user32.NewProc("MonitorFromWindow")
	procMonitorFromPoint    = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procFindWindowExW       = user32.NewProc("FindWindowExW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetDpiForMonitor    = shcore.NewProc("GetDpiForMonitor")
)

const (
	monitorDefaultToNull    = 0
	monitorDefaultToPrimary = 1
	mdtEffectiveDPI         = 0
	swpNoSize               = 0x0001
	swpNoZOrder             = 0x0004
	swpNoActivate           = 0x0010
)

type monitorInfo struct {
	Size    uint32
	Monitor rect32
	Work    rect32
	Flags   uint32
}

// rect32 is Win32's RECT.
type rect32 struct{ Left, Top, Right, Bottom int32 }

// LaunchMonitor is the display the app was launched from. Call it first
// thing in main, before any window of ours exists: at that point the
// foreground window is the one that launched us (Explorer, for the context
// menu and Send to). Without one it falls back to the display under the
// mouse, then the primary display.
func LaunchMonitor() Monitor {
	if hwnd, _, _ := procGetForegroundWindow.Call(); hwnd != 0 {
		if m, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNull); m != 0 {
			return Monitor(m)
		}
	}
	var pt struct{ X, Y int32 }
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); ok != 0 {
		// MonitorFromPoint takes the POINT by value, packed into one argument.
		packed := uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32
		m, _, _ := procMonitorFromPoint.Call(packed, monitorDefaultToPrimary)
		return Monitor(m)
	}
	return 0
}

// Place centres the main window on m at width × height device-independent
// pixels, scaled for m's DPI and shrunk to fit its work area. The window
// should still be hidden. An error means the caller should fall back to
// Wails' own centring.
func Place(m Monitor, width, height int) error {
	if m == 0 {
		return errors.New("no launch monitor")
	}
	hwnd := findMainWindow()
	if hwnd == 0 {
		return errors.New("main window not found")
	}
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, err := procGetMonitorInfoW.Call(uintptr(m), uintptr(unsafe.Pointer(&info))); ok == 0 {
		return err // the display went away since launch
	}
	work := Rect{int(info.Work.Left), int(info.Work.Top), int(info.Work.Right), int(info.Work.Bottom)}

	// Move onto the display first, keeping the size: if its DPI differs,
	// Wails rescales the window in response, and the final size below is
	// computed for the new DPI either way.
	if err := setWindowPos(hwnd, work.Left, work.Top, 0, 0, swpNoSize); err != nil {
		return err
	}
	dpi := monitorDPI(m)
	r := Centre(work, scale(width, dpi), scale(height, dpi))
	return setWindowPos(hwnd, r.Left, r.Top, r.Width(), r.Height(), 0)
}

// findMainWindow returns this process's window of class WindowClass. Other
// instances share the class, so the owning process is checked.
func findMainWindow() uintptr {
	class, _ := windows.UTF16PtrFromString(WindowClass)
	pid := uint32(os.Getpid())
	var hwnd uintptr
	for {
		hwnd, _, _ = procFindWindowExW.Call(0, hwnd, uintptr(unsafe.Pointer(class)), 0)
		if hwnd == 0 {
			return 0
		}
		var owner uint32
		if _, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), &owner); err == nil && owner == pid {
			return hwnd
		}
	}
}

func monitorDPI(m Monitor) int {
	var x, y uint32
	if procGetDpiForMonitor.Find() != nil {
		return 96
	}
	if hr, _, _ := procGetDpiForMonitor.Call(uintptr(m), mdtEffectiveDPI, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y))); hr != 0 || x == 0 {
		return 96
	}
	return int(x)
}

func setWindowPos(hwnd uintptr, x, y, w, h int, flags uintptr) error {
	ok, _, err := procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), flags|swpNoZOrder|swpNoActivate)
	if ok == 0 {
		return err
	}
	return nil
}
