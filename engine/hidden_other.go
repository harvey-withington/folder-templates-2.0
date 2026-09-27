//go:build !windows

package foldertemplate

// hideDir is a no-op off Windows: the leading dot already hides .ft/.
func hideDir(string) {}
