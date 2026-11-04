package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDefaultIcon(t *testing.T) {
	tmpDir := t.TempDir()
	targetIcon := filepath.Join(tmpDir, "icons", "antigravity.png")

	result := EnsureDefaultIcon(targetIcon)
	if result != targetIcon {
		t.Fatalf("expected result %s, got %s", targetIcon, result)
	}

	fi, err := os.Stat(targetIcon)
	if err != nil {
		t.Fatalf("failed to stat written icon: %v", err)
	}
	if fi.Size() < 1000 {
		t.Fatalf("icon size suspiciously small: %d bytes", fi.Size())
	}
}

func TestFindAllAntigravityExecutables(t *testing.T) {
	// Ensure the function executes safely and returns slice without panicking
	targets := FindAllAntigravityExecutables()
	t.Logf("Discovered targets: %d", len(targets))
	for _, target := range targets {
		t.Logf(" - Target: %s, ExePath: %s, IsIDE: %v", target.Name, target.ExePath, target.IsIDE)
	}
}

func TestGenerateWindowsLauncherBatch(t *testing.T) {
	appPath := `C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe`
	proxyURL := "socks5://127.0.0.1:10808"

	batch := GenerateWindowsLauncherBatch(appPath, proxyURL)
	expectedSubstrings := []string{
		`set "ALL_PROXY=socks5h://127.0.0.1:10808"`,
		`set "HTTPS_PROXY=socks5h://127.0.0.1:10808"`,
		`set "HTTP_PROXY=socks5h://127.0.0.1:10808"`,
		`set "SSH_CONNECTION=127.0.0.1 1234 127.0.0.1 22"`,
		`set "SSH_CLIENT=127.0.0.1 1234 22"`,
		`start "" "C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe" --proxy-server=socks5://127.0.0.1:10808 %*`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(batch, sub) {
			t.Errorf("batch output missing expected statement: %q", sub)
		}
	}
}

func TestGenerateWindowsLauncherPowerShell(t *testing.T) {
	appPath := `C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe`
	proxyURL := "socks5://127.0.0.1:10808"

	ps1 := GenerateWindowsLauncherPowerShell(appPath, proxyURL)
	expectedSubstrings := []string{
		`$env:ALL_PROXY = "socks5h://127.0.0.1:10808"`,
		`$env:SSH_CONNECTION = "127.0.0.1 1234 127.0.0.1 22"`,
		`--proxy-server=socks5://127.0.0.1:10808`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(ps1, sub) {
			t.Errorf("powershell output missing expected statement: %q", sub)
		}
	}
}
