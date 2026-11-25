//go:build !windows

package doctor

import "fmt"

func detectWindowsOSInfoNative(arch string) string {
	return fmt.Sprintf("Windows [%s]", arch)
}

func copyToWindowsClipboardNative(_ string) bool {
	return false
}

func openWindowsFileNative(_ string) bool {
	return false
}
