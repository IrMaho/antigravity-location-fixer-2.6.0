//go:build windows

package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndResolveWindowsShortcutNative(t *testing.T) {
	tmpDir := t.TempDir()
	targetExe := filepath.Join(tmpDir, "Antigravity IDE.exe")
	if err := os.WriteFile(targetExe, []byte("MZ"), 0755); err != nil {
		t.Fatalf("failed to create dummy target exe: %v", err)
	}

	lnkPath := filepath.Join(tmpDir, "Antigravity IDE.lnk")
	err := CreateNativeWindowsShortcut(
		lnkPath,
		targetExe,
		"--proxy-server=socks5://127.0.0.1:10808",
		tmpDir,
		targetExe,
		0,
		"Google Antigravity IDE",
		1,
	)
	if err != nil {
		t.Fatalf("CreateNativeWindowsShortcut failed: %v", err)
	}

	resolved := ResolveWindowsShortcut(lnkPath)
	if !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(targetExe)) {
		t.Fatalf("ResolveWindowsShortcut mismatch: got %q, want %q", resolved, targetExe)
	}
}
