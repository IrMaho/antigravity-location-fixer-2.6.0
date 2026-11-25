//go:build windows

package patcher

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func winProtectedPIDs(selfPID int) map[uint32]bool {
	protected := map[uint32]bool{
		0:               true,
		4:               true,
		uint32(selfPID): true,
	}
	if ppid := os.Getppid(); ppid > 0 {
		protected[uint32(ppid)] = true
	}
	if env := strings.TrimSpace(os.Getenv("ANTIGRAVITY_PROTECT_PID")); env != "" {
		var p int
		if _, err := fmt.Sscanf(env, "%d", &p); err == nil && p > 0 {
			protected[uint32(p)] = true
		}
	}
	return protected
}

func queryProcessFullPath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH*4)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func winFindProcessPathBySubstring(targetName string) string {
	cleanTarget := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(targetName), ".exe"))
	if cleanTarget == "" {
		return ""
	}
	selfPID := uint32(os.Getpid())

	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return ""
	}

	for {
		if pe.ProcessID > 4 && pe.ProcessID != selfPID {
			exeName := windows.UTF16ToString(pe.ExeFile[:])
			baseLow := strings.ToLower(strings.TrimSuffix(exeName, ".exe"))
			if strings.Contains(baseLow, cleanTarget) &&
				!strings.Contains(baseLow, "antigravity-cleaner") &&
				!strings.Contains(baseLow, "antigravitylocationfixer") &&
				!strings.Contains(baseLow, "antigravity-gui") {
				if full := queryProcessFullPath(pe.ProcessID); full != "" {
					if fi, stErr := os.Stat(full); stErr == nil && !fi.IsDir() {
						return full
					}
				}
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return ""
}

func winFindRunningLanguageServerPID(selfPID int) int {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return 0
	}

	for {
		if pe.ProcessID > 4 && int(pe.ProcessID) != selfPID {
			low := strings.ToLower(windows.UTF16ToString(pe.ExeFile[:]))
			if strings.Contains(low, "language_server") || low == "agy.exe" {
				return int(pe.ProcessID)
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return 0
}

func winIsAnyProcessRunning(exactNames []string, selfPID int) bool {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snap)

	targetSet := make(map[string]bool, len(exactNames))
	for _, n := range exactNames {
		targetSet[strings.ToLower(n)] = true
	}

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return false
	}

	for {
		if pe.ProcessID > 4 && int(pe.ProcessID) != selfPID {
			low := strings.ToLower(windows.UTF16ToString(pe.ExeFile[:]))
			if targetSet[low] || strings.HasPrefix(low, "language_server") {
				return true
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return false
}

func winTerminateProcessesByNames(exactNames []string, matchLanguageServerPrefix bool, selfPID int) int {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snap)

	protected := winProtectedPIDs(selfPID)
	targetSet := make(map[string]bool, len(exactNames))
	for _, n := range exactNames {
		targetSet[strings.ToLower(n)] = true
	}

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return 0
	}

	terminated := 0
	for {
		if !protected[pe.ProcessID] {
			low := strings.ToLower(windows.UTF16ToString(pe.ExeFile[:]))
			matched := targetSet[low] || (matchLanguageServerPrefix && strings.HasPrefix(low, "language_server") && strings.HasSuffix(low, ".exe"))
			if matched {
				h, openErr := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pe.ProcessID)
				if openErr == nil {
					if windows.TerminateProcess(h, 1) == nil {
						terminated++
					}
					windows.CloseHandle(h)
				}
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return terminated
}
