package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInstallerPath(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Installer filenames
	installerNames := []string{
		"Antigravity-x64 (1).exe",
		"Antigravity-Setup-x64.exe",
		"antigravity-installer.exe",
		"Antigravity-x64.exe",
	}

	for _, name := range installerNames {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, []byte("fake binary"), 0755); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if !IsInstallerPath(p) {
			t.Errorf("expected IsInstallerPath(%q) to be true, got false", name)
		}
	}

	// 2. Real installed application (has resources/app or resources/bin/language_server.exe nearby)
	installedDir := filepath.Join(tmpDir, "InstalledApp")
	resDir := filepath.Join(installedDir, "resources", "app")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		t.Fatalf("failed to create resources dir: %v", err)
	}
	appExe := filepath.Join(installedDir, "Antigravity IDE.exe")
	if err := os.WriteFile(appExe, []byte("fake installed exe"), 0755); err != nil {
		t.Fatalf("failed to create appExe: %v", err)
	}

	if IsInstallerPath(appExe) {
		t.Errorf("expected IsInstallerPath(%q) for real installed app to be false, got true", appExe)
	}
}
