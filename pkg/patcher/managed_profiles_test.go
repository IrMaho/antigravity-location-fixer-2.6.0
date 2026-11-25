package patcher

import (
	"testing"
)

func TestFindAllManagedProfiles(t *testing.T) {
	profiles := FindAllManagedProfiles()
	t.Logf("Discovered %d managed profiles", len(profiles))
	for _, p := range profiles {
		t.Logf("Profile: %s, Path: %s, AgyBin: %s, IsPatched: %v", p.Name, p.Path, p.AgyBin, p.IsPatched)
	}
	if len(profiles) == 0 {
		t.Log("No managed profiles discovered on this machine (might be clean environment)")
	}
}

func TestCleanerConfig(t *testing.T) {
	cfg := LoadCleanerConfig()
	if cfg.ProfileScope != "all" && cfg.ProfileScope != "primary" {
		t.Fatalf("unexpected profile scope: %s", cfg.ProfileScope)
	}
	t.Logf("Current ProfileScope: %s", cfg.ProfileScope)
}

func TestPatchSpecificManagedProfile(t *testing.T) {
	devshammdAgy := `C:\Antigravity-Profiles\devshammd\home\.gemini\bin\agy.exe`
	if !fileExists(devshammdAgy) {
		t.Skip("devshammd agy.exe not found")
	}
	t.Logf("Checking devshammd before patch: IsAgyPatched = %v", IsAgyPatched(devshammdAgy))
	res, err := PatchAgy(devshammdAgy)
	if err != nil {
		t.Fatalf("PatchAgy failed: %v", err)
	}
	t.Logf("Patch result: %s", res)
	if !IsAgyPatched(devshammdAgy) {
		t.Fatalf("devshammd is still unpatched after PatchAgy!")
	}
	t.Logf("Verification SUCCESS: devshammd is now patched!")
}
