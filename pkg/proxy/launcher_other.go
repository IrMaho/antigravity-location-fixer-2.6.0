//go:build !windows

package proxy

// FindAntigravityExecutablesFromRegistry is a no-op on non-Windows platforms.
func FindAntigravityExecutablesFromRegistry() []AppTarget {
	return nil
}

// ResolveWindowsShortcut is a no-op on non-Windows platforms.
func ResolveWindowsShortcut(lnkPath string) string {
	return lnkPath
}
