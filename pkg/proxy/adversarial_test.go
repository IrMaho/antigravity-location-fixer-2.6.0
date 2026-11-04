package proxy

import (
	"fmt"
	"strings"
	"testing"
)

func TestAdv_WindowsLauncherPathsWithSpacesAndSpecialChars(t *testing.T) {
	appPathWithSpaces := `C:\Program Files (x86)\Antigravity & IDE\Antigravity IDE.exe`
	proxyURL := "socks5://127.0.0.1:10808"

	// 1. Test Batch launcher generator
	batch := GenerateWindowsLauncherBatch(appPathWithSpaces, proxyURL)
	t.Logf("Batch launcher snippet:\n%s", batch)

	if !strings.Contains(batch, fmt.Sprintf(`"%s"`, appPathWithSpaces)) {
		t.Errorf("batch launcher did not properly quote app path with spaces")
	}

	// Verify all environment variables present
	requiredEnv := []string{
		`set "ALL_PROXY=socks5h://127.0.0.1:10808"`,
		`set "HTTPS_PROXY=socks5h://127.0.0.1:10808"`,
		`set "HTTP_PROXY=socks5h://127.0.0.1:10808"`,
		`set "SSH_CONNECTION=127.0.0.1 1234 127.0.0.1 22"`,
		`set "SSH_CLIENT=127.0.0.1 1234 22"`,
	}
	for _, env := range requiredEnv {
		if !strings.Contains(batch, env) {
			t.Errorf("batch missing required env: %s", env)
		}
	}

	// 2. Test PowerShell launcher generator
	ps1 := GenerateWindowsLauncherPowerShell(appPathWithSpaces, proxyURL)
	t.Logf("PowerShell launcher snippet:\n%s", ps1)

	if !strings.Contains(ps1, fmt.Sprintf(`Start-Process -FilePath "%s"`, appPathWithSpaces)) {
		t.Errorf("powershell launcher did not properly quote app path with spaces")
	}
}

func TestAdv_WindowsLauncherHttpProxy(t *testing.T) {
	appPath := `C:\Users\Test\Antigravity IDE.exe`
	proxyURL := "http://127.0.0.1:8080"

	batch := GenerateWindowsLauncherBatch(appPath, proxyURL)
	if !strings.Contains(batch, `set "ALL_PROXY=http://127.0.0.1:8080"`) {
		t.Errorf("batch did not configure ALL_PROXY for http proxy: %s", batch)
	}
	if !strings.Contains(batch, `--proxy-server=http://127.0.0.1:8080`) {
		t.Errorf("batch did not pass --proxy-server=http://127.0.0.1:8080")
	}
}
