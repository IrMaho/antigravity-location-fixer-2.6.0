//go:build windows

package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// FindAntigravityExecutablesFromRegistry scans the Windows Registry uninstall keys
// for official Google Antigravity and Antigravity IDE installations.
func FindAntigravityExecutablesFromRegistry() []AppTarget {
	var targets []AppTarget
	seen := make(map[string]bool)

	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, k := range keys {
		key, err := registry.OpenKey(k.root, k.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}

		for _, sub := range subkeys {
			sk, err := registry.OpenKey(k.root, k.path+`\`+sub, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			disp, _, _ := sk.GetStringValue("DisplayName")
			loc, _, _ := sk.GetStringValue("InstallLocation")
			icon, _, _ := sk.GetStringValue("DisplayIcon")
			sk.Close()

			dispLower := strings.ToLower(disp)
			if !strings.Contains(dispLower, "antigravity") {
				continue
			}

			isIDE := strings.Contains(dispLower, "ide")

			// 1. Check DisplayIcon
			if icon != "" {
				// Strip icon index like ",0"
				if idx := strings.Index(icon, ","); idx != -1 {
					icon = icon[:idx]
				}
				icon = strings.Trim(icon, `"`)
				cleanIcon := filepath.Clean(icon)
				if fi, err := os.Stat(cleanIcon); err == nil && !fi.IsDir() && !seen[cleanIcon] {
					seen[cleanIcon] = true
					name := "Antigravity"
					if isIDE || strings.Contains(strings.ToLower(cleanIcon), "ide") {
						name = "Antigravity IDE"
						isIDE = true
					}
					targets = append(targets, AppTarget{
						Name:    name,
						ExePath: cleanIcon,
						IsIDE:   isIDE,
					})
				}
			}

			// 2. Check InstallLocation
			if loc != "" {
				loc = strings.Trim(loc, `"`)
				for _, exeName := range []string{"Antigravity IDE.exe", "Antigravity.exe", "antigravity.exe"} {
					exe := filepath.Join(loc, exeName)
					cleanExe := filepath.Clean(exe)
					if fi, err := os.Stat(cleanExe); err == nil && !fi.IsDir() && !seen[cleanExe] {
						seen[cleanExe] = true
						name := "Antigravity"
						targetIsIDE := isIDE
						if strings.Contains(strings.ToLower(exeName), "ide") {
							name = "Antigravity IDE"
							targetIsIDE = true
						}
						targets = append(targets, AppTarget{
							Name:    name,
							ExePath: cleanExe,
							IsIDE:   targetIsIDE,
						})
					}
				}
			}
		}
	}

	return targets
}

// ResolveWindowsShortcut resolves a .lnk file to its actual target path.
func ResolveWindowsShortcut(lnkPath string) string {
	if !strings.HasSuffix(strings.ToLower(lnkPath), ".lnk") {
		return lnkPath
	}
	escaped := strings.ReplaceAll(lnkPath, "'", "''")
	script := fmt.Sprintf(`$sh = New-Object -ComObject WScript.Shell; $sc = $sh.CreateShortcut('%s'); [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Write-Output $sc.TargetPath`, escaped)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	if err == nil {
		target := strings.TrimSpace(string(out))
		if target != "" {
			if _, err := os.Stat(target); err == nil {
				return target
			}
		}
	}
	return lnkPath
}
