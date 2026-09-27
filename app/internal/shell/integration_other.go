//go:build !windows

package shell

import "errors"

// Integration is Windows-only; elsewhere every feature reports off.
type Integration struct{}

// Default returns an Integration that does nothing.
func Default() (*Integration, error) { return &Integration{}, nil }

// For returns an Integration that does nothing.
func For(string) (*Integration, error) { return &Integration{}, nil }

// Status reports every feature off.
func (in *Integration) Status() map[Feature]bool {
	out := map[Feature]bool{}
	for _, f := range Features {
		out[f] = false
	}
	return out
}

// Set is unsupported off Windows.
func (in *Integration) Set(Feature, bool) error {
	return errors.New("shell integration is only available on Windows")
}

// RemoveAll does nothing off Windows.
func (in *Integration) RemoveAll() error { return nil }
