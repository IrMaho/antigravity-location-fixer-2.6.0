package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLauncher_SpecialCharactersAndProxies stress tests GenerateWindowsLauncherBatch
// and GenerateWindowsLauncherPowerShell against various proxy URLs and application paths.
func TestLauncher_SpecialCharactersAndProxies(t *testing.T) {
	testCases := []struct {
		name     string
		appPath  string
		proxyURL string
	}{
		{
			name:     "standard_ide_path_with_spaces",
			appPath:  `C:\Program Files (x86)\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "socks5://127.0.0.1:1080",
		},
		{
			name:     "socks5h_already_prefixed",
			appPath:  `C:\Users\ASUS\AppData\Local\Programs\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "socks5h://127.0.0.1:10808",
		},
		{
			name:     "http_proxy_with_port",
			appPath:  `C:\Users\User Name\Antigravity.exe`,
			proxyURL: "http://proxy.internal:8080",
		},
		{
			name:     "https_proxy_with_auth",
			appPath:  `C:\Tools\Antigravity\antigravity.exe`,
			proxyURL: "http://admin:secret123@proxy.corp.net:3128",
		},
		{
			name:     "proxy_with_dollar_sign_credential",
			appPath:  `C:\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "http://user:$uper$ecret@127.0.0.1:8080",
		},
		{
			name:     "proxy_with_ampersand_credential",
			appPath:  `C:\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "http://user:foo&bar@127.0.0.1:8080",
		},
		{
			name:     "proxy_with_percent_encoding",
			appPath:  `C:\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "socks5://user%40domain:pass%21@127.0.0.1:1080",
		},
		{
			name:     "empty_proxy_url",
			appPath:  `C:\Users\ASUS\Antigravity.exe`,
			proxyURL: "",
		},
		{
			name:     "app_path_with_parentheses",
			appPath:  `C:\Program Files (x86)\Antigravity (Preview)\Antigravity.exe`,
			proxyURL: "socks5://127.0.0.1:1080",
		},
	}

	for _, tc := range testCases {
		t.Run("Batch_"+tc.name, func(t *testing.T) {
			batch := GenerateWindowsLauncherBatch(tc.appPath, tc.proxyURL)
			if batch == "" {
				t.Fatalf("empty batch script generated")
			}

			// Empirical check 1: Batch syntax validity with cmd.exe
			tmpDir := t.TempDir()
			batFile := filepath.Join(tmpDir, "test_launcher.bat")

			// To empirically verify syntax without actually starting a GUI process,
			// we create a dummy mock executable or inspect cmd parsing.
			// Let's create an echo script or test environment export.
			verifyBatContent := strings.Replace(batch, "start \"\" \""+tc.appPath+"\"", "rem start command mocked\nrem", 1)
			verifyBatContent += fmt.Sprintf("\r\necho ALL_PROXY=%%ALL_PROXY%%\r\necho HTTPS_PROXY=%%HTTPS_PROXY%%\r\necho SSH_CONNECTION=%%SSH_CONNECTION%%\r\n")

			if err := os.WriteFile(batFile, []byte(verifyBatContent), 0755); err != nil {
				t.Fatalf("failed to write bat file: %v", err)
			}

			cmd := exec.Command("cmd.exe", "/c", batFile)
			out, err := cmd.CombinedOutput()
			outStr := string(out)
			t.Logf("cmd output for %s:\n%s", tc.name, outStr)
			if err != nil {
				t.Errorf("cmd execution failed for %s: %v, output: %s", tc.name, err, outStr)
			}

			// Verify environment variable propagation
			if !strings.Contains(outStr, "SSH_CONNECTION=127.0.0.1 1234 127.0.0.1 22") {
				t.Errorf("SSH_CONNECTION was not set properly in cmd: %s", outStr)
			}
		})

		t.Run("PowerShell_"+tc.name, func(t *testing.T) {
			ps1 := GenerateWindowsLauncherPowerShell(tc.appPath, tc.proxyURL)
			if ps1 == "" {
				t.Fatalf("empty powershell script generated")
			}

			// Empirical check 2: PowerShell syntax validation using PowerShell parser
			tmpDir := t.TempDir()
			ps1File := filepath.Join(tmpDir, "test_launcher.ps1")
			if err := os.WriteFile(ps1File, []byte(ps1), 0755); err != nil {
				t.Fatalf("failed to write ps1 file: %v", err)
			}

			// Run PowerShell syntax check via AST parser
			psCheckCmd := fmt.Sprintf(`
$errors = $null
[System.Management.Automation.Language.Parser]::ParseFile("%s", [ref]$null, [ref]$errors)
if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Error $_.Message }
    exit 1
}
`, ps1File)

			cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCheckCmd)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("PowerShell syntax error for %s: %v, output: %s", tc.name, err, string(out))
			}

			// Empirical check 3: Variable expansion vulnerability in PowerShell
			// If proxyURL contains '$' e.g. "http://user:$uper$ecret@127.0.0.1:8080",
			// PowerShell double quotes "$env:ALL_PROXY = \"%s\"" will interpolate $uper and $ecret!
			if strings.Contains(tc.proxyURL, "$") {
				// Let's run PowerShell and check what value $env:ALL_PROXY actually receives!
				evalScript := fmt.Sprintf(`
%s
Write-Output "ACTUAL_PROXY=$env:ALL_PROXY"
`, strings.Replace(ps1, "Start-Process", "# Start-Process", 1))

				evalFile := filepath.Join(tmpDir, "eval.ps1")
				_ = os.WriteFile(evalFile, []byte(evalScript), 0755)

				runCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", evalFile)
				evalOut, evalErr := runCmd.CombinedOutput()
				evalStr := string(evalOut)
				t.Logf("PowerShell eval output for %s: %s (err: %v)", tc.name, evalStr, evalErr)

				// Check if the dollar signs were lost due to string interpolation
				if strings.Contains(evalStr, "ACTUAL_PROXY=") {
					if !strings.Contains(evalStr, "$uper$ecret") {
						t.Errorf("BUG DISCOVERED: PowerShell double quotes interpolated and stripped dollar-sign variables! Got: %s, expected password with $uper$ecret", evalStr)
					}
				}
			}
		})
	}
}

// TestLauncher_CommandInjectionStress checks whether malicious or specially formatted
// paths and proxy URLs can break out of quotes or cause unexpected command execution.
func TestLauncher_CommandInjectionStress(t *testing.T) {
	attackCases := []struct {
		name     string
		appPath  string
		proxyURL string
	}{
		{
			name:     "ampersand_in_proxy_batch",
			appPath:  `C:\Antigravity IDE\Antigravity IDE.exe`,
			proxyURL: "socks5://127.0.0.1:1080&calc.exe",
		},
		{
			name:     "quote_breakout_apppath_batch",
			appPath:  `C:\Antigravity IDE\" & echo EXPLOIT & rem \Antigravity.exe`,
			proxyURL: "socks5://127.0.0.1:1080",
		},
		{
			name:     "quote_breakout_powershell",
			appPath:  `C:\Antigravity IDE\"; Write-Output 'INJECTED'; rem "`,
			proxyURL: "socks5://127.0.0.1:1080",
		},
		{
			name:     "backtick_in_powershell",
			appPath:  `C:\Antigravity IDE` + "`" + `a\app.exe`,
			proxyURL: "socks5://127.0.0.1:1080",
		},
	}

	for _, ac := range attackCases {
		t.Run("Batch_"+ac.name, func(t *testing.T) {
			batch := GenerateWindowsLauncherBatch(ac.appPath, ac.proxyURL)
			// Check if batch handles quotes or escapes
			t.Logf("Batch script for %s:\n%s", ac.name, batch)
		})

		t.Run("PowerShell_"+ac.name, func(t *testing.T) {
			ps1 := GenerateWindowsLauncherPowerShell(ac.appPath, ac.proxyURL)
			t.Logf("PowerShell script for %s:\n%s", ac.name, ps1)

			tmpDir := t.TempDir()
			ps1File := filepath.Join(tmpDir, "attack.ps1")
			_ = os.WriteFile(ps1File, []byte(ps1), 0755)

			psCheckCmd := fmt.Sprintf(`
$errors = $null
[System.Management.Automation.Language.Parser]::ParseFile("%s", [ref]$null, [ref]$errors)
if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Output "PARSER_ERROR: $($_.Message)" }
}
`, ps1File)

			cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCheckCmd)
			out, _ := cmd.CombinedOutput()
			t.Logf("Parser output for %s: %s", ac.name, string(out))
		})
	}
}
