//go:build !windows

package gui

func browseFileWindowsNative() (string, error) {
	return "", nil
}

func browseFolderWindowsNative() (string, error) {
	return "", nil
}
