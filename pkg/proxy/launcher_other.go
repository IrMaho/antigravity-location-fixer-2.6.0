//go:build !windows

package proxy

import "fmt"

// FindAntigravityExecutablesFromRegistry is a no-op on non-Windows platforms.
func FindAntigravityExecutablesFromRegistry() []AppTarget {
	return nil
}

// ResolveWindowsShortcut is a no-op on non-Windows platforms.
func ResolveWindowsShortcut(lnkPath string) string {
	return lnkPath
}

// CreateNativeWindowsShortcut is unsupported on non-Windows platforms.
func CreateNativeWindowsShortcut(_, _, _, _, _ string, _ int, _ string, _ int) error {
	return fmt.Errorf("native Windows shortcut creation requires Windows")
}

func launchElevatedPlatform(_ string, _ []string) error {
	return fmt.Errorf("Windows elevation requires Windows")
}
