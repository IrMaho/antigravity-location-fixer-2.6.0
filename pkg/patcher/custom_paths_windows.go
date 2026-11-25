//go:build windows

package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	regDriveCacheMu   sync.Mutex
	regDriveCacheDirs []string
	regDriveCacheAt   time.Time
)

func getMountedFixedAndRemovableDrives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var drives []string
	for i := 0; i < 26; i++ {
		if (mask & (1 << uint(i))) != 0 {
			root := string(rune('A'+i)) + `:\`
			ptr, err := syscall.UTF16PtrFromString(root)
			if err != nil {
				continue
			}
			dtype := windows.GetDriveType(ptr)
			if dtype == windows.DRIVE_REMOVABLE || dtype == windows.DRIVE_FIXED {
				drives = append(drives, root)
			}
		}
	}
	return drives
}

func findRegistryAndDriveInstallDirs() []string {
	regDriveCacheMu.Lock()
	if time.Since(regDriveCacheAt) < 30*time.Second && regDriveCacheDirs != nil {
		out := make([]string, len(regDriveCacheDirs))
		copy(out, regDriveCacheDirs)
		regDriveCacheMu.Unlock()
		return out
	}
	regDriveCacheMu.Unlock()

	dirs := scanRegistryAndDriveInstallDirsUncached()

	regDriveCacheMu.Lock()
	regDriveCacheDirs = make([]string, len(dirs))
	copy(regDriveCacheDirs, dirs)
	regDriveCacheAt = time.Now()
	regDriveCacheMu.Unlock()

	return dirs
}

func scanRegistryAndDriveInstallDirsUncached() []string {
	var dirs []string
	seen := make(map[string]bool)
	addDir := func(p string) {
		if p == "" {
			return
		}
		clean := filepath.Clean(p)
		key := strings.ToLower(clean)
		if !seen[key] {
			if fi, err := os.Stat(clean); err == nil && fi.IsDir() {
				seen[key] = true
				dirs = append(dirs, clean)
			}
		}
	}

	// 1. Windows Registry Uninstall keys
	regRoots := []struct {
		hive registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, kr := range regRoots {
		key, err := registry.OpenKey(kr.hive, kr.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}
		for _, sub := range subkeys {
			sk, err := registry.OpenKey(kr.hive, kr.path+`\`+sub, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			disp, _, _ := sk.GetStringValue("DisplayName")
			loc, _, _ := sk.GetStringValue("InstallLocation")
			icon, _, _ := sk.GetStringValue("DisplayIcon")
			sk.Close()

			if !strings.Contains(strings.ToLower(sub+" "+disp), "antigravity") {
				continue
			}
			if loc != "" {
				addDir(strings.Trim(loc, `"`))
			}
			if icon != "" {
				if idx := strings.Index(icon, ","); idx != -1 {
					icon = icon[:idx]
				}
				icon = strings.TrimSpace(strings.Trim(icon, `"`))
				if icon != "" {
					addDir(filepath.Dir(icon))
				}
			}
		}
	}

	// 2. Mounted drive roots (standard install folders + 1-level shallow scan)
	appNames := []string{"Antigravity IDE", "Antigravity", "antigravity"}
	username := strings.TrimSpace(os.Getenv("USERNAME"))

	for _, drv := range getMountedFixedAndRemovableDrives() {
		for _, appName := range appNames {
			addDir(filepath.Join(drv, appName))
			addDir(filepath.Join(drv, "Programs", appName))
			addDir(filepath.Join(drv, "Program Files", appName))
			addDir(filepath.Join(drv, "Program Files (x86)", appName))
			addDir(filepath.Join(drv, "Apps", appName))
			addDir(filepath.Join(drv, "Tools", appName))
			addDir(filepath.Join(drv, "Software", appName))
			if username != "" {
				addDir(filepath.Join(drv, "Users", username, "AppData", "Local", "Programs", appName))
			}
		}

		if entries, err := os.ReadDir(drv); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				low := strings.ToLower(e.Name())
				if strings.HasPrefix(low, "$") || strings.HasPrefix(low, ".") || low == "windows" || low == "system volume information" {
					continue
				}
				topPath := filepath.Join(drv, e.Name())
				if strings.Contains(low, "antigravity") {
					addDir(topPath)
				}
				for _, appName := range []string{"Antigravity IDE", "Antigravity"} {
					addDir(filepath.Join(topPath, appName))
				}
			}
		}
	}

	return dirs
}
