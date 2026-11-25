package patcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCustomAndDriveInstallDirs_FromEnvAndSettings(t *testing.T) {
	tmp := t.TempDir()

	// 1. Custom IDE via ANTIGRAVITY_IDE_EXE env var
	envInstall := filepath.Join(tmp, "CustomEnv", "Antigravity IDE")
	envMainJs := filepath.Join(envInstall, "resources", "app", "out", "main.js")
	envExtJs := filepath.Join(envInstall, "resources", "app", "extensions", "agy", "vscode", "extension.js")
	if err := os.MkdirAll(filepath.Dir(envMainJs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(envExtJs), 0o755); err != nil {
		t.Fatal(err)
	}
	envExe := filepath.Join(envInstall, "Antigravity IDE.exe")
	if err := os.WriteFile(envExe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envMainJs, []byte("resetIsTierGCPTos(),this.isGoogleInternal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envExtJs, []byte("// ext"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Custom IDE via settings.json ide_exe_path
	cfgInstall := filepath.Join(tmp, "DriveD", "Programs", "Antigravity IDE")
	cfgMainJs := filepath.Join(cfgInstall, "resources", "app", "out", "main.js")
	if err := os.MkdirAll(filepath.Dir(cfgMainJs), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgExe := filepath.Join(cfgInstall, "Antigravity IDE.exe")
	if err := os.WriteFile(cfgExe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgMainJs, []byte("main"), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(tmp, "settings.json")
	settingsBody := `{"theme":"Dark","ide_exe_path":"` + filepath.ToSlash(cfgExe) + `"}`
	if err := os.WriteFile(settingsPath, []byte(settingsBody), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ANTIGRAVITY_IDE_EXE", envExe)
	t.Setenv("ANTIGRAVITY_MANAGER_SETTINGS", settingsPath)

	roots := FindCustomAndDriveInstallDirs()
	foundEnv := false
	foundCfg := false
	for _, r := range roots {
		if filepath.Clean(r) == filepath.Clean(envInstall) {
			foundEnv = true
		}
		if filepath.Clean(r) == filepath.Clean(cfgInstall) {
			foundCfg = true
		}
	}
	if !foundEnv {
		t.Fatalf("expected env install root %s in %v", envInstall, roots)
	}
	if !foundCfg {
		t.Fatalf("expected settings.json install root %s in %v", cfgInstall, roots)
	}

	allMain := FindAllIdeMainJs()
	hasEnvMain := false
	hasCfgMain := false
	for _, m := range allMain {
		if filepath.Clean(m) == filepath.Clean(envMainJs) {
			hasEnvMain = true
		}
		if filepath.Clean(m) == filepath.Clean(cfgMainJs) {
			hasCfgMain = true
		}
	}
	if !hasEnvMain || !hasCfgMain {
		t.Fatalf("expected custom main.js paths in FindAllIdeMainJs(): got %v", allMain)
	}
}
