//go:build windows

package foldertemplate

import "syscall"

// hideDir sets the Hidden attribute on path, as the C# app did for .ft/.
// Best effort: a failure leaves the folder visible, which is harmless.
func hideDir(path string) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil || attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
		return
	}
	_ = syscall.SetFileAttributes(p, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
}
