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
		`$env:ALL_PROXY = 'socks5h://127.0.0.1:10808'`,
		`$env:SSH_CONNECTION = "127.0.0.1 1234 127.0.0.1 22"`,
		`--proxy-server=socks5://127.0.0.1:10808`,
		`Start-Process -FilePath "C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe" -ArgumentList $appArgs`,
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(ps1, sub) {
			t.Errorf("powershell output missing expected statement: %q", sub)
		}
	}
}

func TestBuildDirectEnv(t *testing.T) {
	baseEnv := []string{
		"HTTP_PROXY=http://127.0.0.1:8080",
		"HTTPS_PROXY=http://127.0.0.1:8080",
		"ALL_PROXY=socks5://127.0.0.1:10808",
		"JETSKI_OAUTH_TOKEN=secret_token",
		"PATH=/usr/bin:/bin",
	}

	clean := BuildDirectEnv(baseEnv)
	var foundProxy, foundSshConn, foundSshClient, foundPath, foundLeak bool
	for _, e := range clean {
		if strings.HasPrefix(e, "HTTP_PROXY=") || strings.HasPrefix(e, "HTTPS_PROXY=") || strings.HasPrefix(e, "ALL_PROXY=") {
			foundProxy = true
		}
		if strings.HasPrefix(e, "JETSKI_OAUTH_TOKEN=") {
			foundLeak = true
		}
		if e == "SSH_CONNECTION=127.0.0.1 1234 127.0.0.1 22" {
			foundSshConn = true
		}
		if e == "SSH_CLIENT=127.0.0.1 1234 22" {
			foundSshClient = true
		}
		if e == "PATH=/usr/bin:/bin" {
			foundPath = true
		}
	}

	if foundProxy {
		t.Errorf("BuildDirectEnv failed to strip proxy environment variables: %v", clean)
	}
	if foundLeak {
		t.Errorf("BuildDirectEnv leaked sensitive variable: %v", clean)
	}
	if !foundSshConn || !foundSshClient {
		t.Errorf("BuildDirectEnv did not inject SSH spoofing variables: %v", clean)
	}
	if !foundPath {
		t.Errorf("BuildDirectEnv lost original PATH: %v", clean)
	}
}

func TestGenerateWindowsLauncherBatch_Direct(t *testing.T) {
	appPath := `C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe`
	batch := GenerateWindowsLauncherBatch(appPath, "direct://")

	if strings.Contains(batch, "--proxy-server") {
		t.Errorf("Direct batch launcher should not contain --proxy-server: %s", batch)
	}
	if !strings.Contains(batch, `set "ALL_PROXY="`) {
		t.Errorf("Direct batch launcher should clear ALL_PROXY: %s", batch)
	}
	if !strings.Contains(batch, `set "SSH_CONNECTION=127.0.0.1 1234 127.0.0.1 22"`) {
		t.Errorf("Direct batch launcher should preserve SSH_CONNECTION: %s", batch)
	}
	if !strings.Contains(batch, `start "" "C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe" %*`) {
		t.Errorf("Direct batch launcher missing start command: %s", batch)
	}
}

func TestGenerateWindowsLauncherPowerShell_Direct(t *testing.T) {
	appPath := `C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe`
	ps1 := GenerateWindowsLauncherPowerShell(appPath, "direct://")

	if strings.Contains(ps1, "--proxy-server") {
		t.Errorf("Direct powershell launcher should not contain --proxy-server: %s", ps1)
	}
	if !strings.Contains(ps1, "Remove-Item Env:ALL_PROXY") {
		t.Errorf("Direct powershell launcher should remove ALL_PROXY: %s", ps1)
	}
	if !strings.Contains(ps1, `$env:SSH_CONNECTION = "127.0.0.1 1234 127.0.0.1 22"`) {
		t.Errorf("Direct powershell launcher should set SSH_CONNECTION: %s", ps1)
	}
	if !strings.Contains(ps1, "-Verb RunAs") {
		t.Errorf("Direct powershell launcher should include elevation fallback: %s", ps1)
	}
}
