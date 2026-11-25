//go:build !windows

package patcher

func winFindProcessPathBySubstring(_ string) string {
	return ""
}

func winFindRunningLanguageServerPID(_ int) int {
	return 0
}

func winIsAnyProcessRunning(_ []string, _ int) bool {
	return false
}

func winTerminateProcessesByNames(_ []string, _ bool, _ int) int {
	return 0
}
