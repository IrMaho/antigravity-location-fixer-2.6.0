package patcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type agyPatchCacheEntry struct {
	size    int64
	modNano int64
	patched bool
}

var (
	agyPatchCacheMu sync.Mutex
	agyPatchCache   = make(map[string]agyPatchCacheEntry)
)

type managerSettings struct {
	IdeExePath string `json:"ide_exe_path"`
}

// resolveInstallRootsFromPath normalizes a user-configured file or folder path into
// candidate Antigravity install root directories.
func resolveInstallRootsFromPath(raw string) []string {
	cleaned := strings.TrimSpace(strings.Trim(raw, `"'`))
	if cleaned == "" {
		return nil
	}

	var roots []string
	seen := make(map[string]bool)
	addRoot := func(dir string) {
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		key := strings.ToLower(clean)
		if !seen[key] {
			if fi, err := os.Stat(clean); err == nil && fi.IsDir() {
				seen[key] = true
				roots = append(roots, clean)
			}
		}
	}

	if fi, err := os.Stat(cleaned); err == nil {
		if !fi.IsDir() {
			cur := filepath.Dir(cleaned)
			addRoot(cur)
			for i := 0; i < 4; i++ {
				parent := filepath.Dir(cur)
				if parent == cur || parent == "" {
					break
				}
				cur = parent
				addRoot(cur)
			}
		} else {
			addRoot(cleaned)
			for _, sub := range []string{"Antigravity IDE", "Antigravity"} {
				addRoot(filepath.Join(cleaned, sub))
			}
			cur := cleaned
			for i := 0; i < 3; i++ {
				parent := filepath.Dir(cur)
				if parent == cur || parent == "" {
					break
				}
				cur = parent
				addRoot(cur)
			}
		}
	}
	return roots
}

// FindCustomAndDriveInstallDirs returns Antigravity install roots discovered from
// ANTIGRAVITY_IDE_EXE, AntigravityManager/settings.json, Windows Registry, and mounted drives.
func FindCustomAndDriveInstallDirs() []string {
	var roots []string
	seen := make(map[string]bool)
	add := func(p string) {
		if p == "" {
			return
		}
		clean := filepath.Clean(p)
		key := strings.ToLower(clean)
		if !seen[key] {
			seen[key] = true
			roots = append(roots, clean)
		}
	}

	if envExe := strings.TrimSpace(os.Getenv("ANTIGRAVITY_IDE_EXE")); envExe != "" {
		for _, r := range resolveInstallRootsFromPath(envExe) {
			add(r)
		}
	}

	for _, sFile := range findManagerSettingsCandidates() {
		data, err := os.ReadFile(sFile)
		if err != nil {
			continue
		}
		var cfg managerSettings
		if json.Unmarshal(data, &cfg) == nil && strings.TrimSpace(cfg.IdeExePath) != "" {
			for _, r := range resolveInstallRootsFromPath(cfg.IdeExePath) {
				add(r)
			}
		}
	}

	for _, r := range findRegistryAndDriveInstallDirs() {
		add(r)
	}

	return roots
}

func findManagerSettingsCandidates() []string {
	var cands []string
	if envSettings := strings.TrimSpace(os.Getenv("ANTIGRAVITY_MANAGER_SETTINGS")); envSettings != "" {
		cands = append(cands, envSettings)
	}
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		cands = append(cands, filepath.Join(localApp, "AntigravityManager", "settings.json"))
	}
	if user := os.Getenv("USERNAME"); user != "" {
		sysDrive := os.Getenv("SystemDrive")
		if sysDrive == "" {
			sysDrive = "C:"
		}
		cands = append(cands, filepath.Join(sysDrive+string(os.PathSeparator), "Users", user, "AppData", "Local", "AntigravityManager", "settings.json"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cands = append(cands, filepath.Join(home, "AppData", "Local", "AntigravityManager", "settings.json"))
	}
	return cands
}

// ManagedProfile describes an Antigravity Manager isolated profile
type ManagedProfile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	AgyBin    string `json:"agy_bin,omitempty"`
	IsPatched bool   `json:"is_patched"`
}

// CleanerConfig stores user preferences such as profile scope ("all" or "primary")
type CleanerConfig struct {
	ProfileScope string `json:"profile_scope"` // "all" or "primary"
}

// GetCleanerConfigPath returns the persistent config file path for Antigravity Cleaner
func GetCleanerConfigPath() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		home = GetRealUserHomeWindows()
	}
	return filepath.Join(home, ".config", "antigravity-cleaner", "config.json")
}

// LoadCleanerConfig loads the cleaner configuration, defaulting ProfileScope to "all"
func LoadCleanerConfig() CleanerConfig {
	cfg := CleanerConfig{ProfileScope: "all"}
	p := GetCleanerConfigPath()
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg.ProfileScope != "primary" {
		cfg.ProfileScope = "all"
	}
	return cfg
}

// SaveCleanerConfig persists the cleaner configuration
func SaveCleanerConfig(cfg CleanerConfig) error {
	p := GetCleanerConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func findManagerRegistryCandidates() []string {
	var cands []string
	if envReg := strings.TrimSpace(os.Getenv("ANTIGRAVITY_MANAGER_REGISTRY")); envReg != "" {
		cands = append(cands, envReg)
	}
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		cands = append(cands, filepath.Join(localApp, "AntigravityManager", "profiles_registry.json"))
	}
	if user := os.Getenv("USERNAME"); user != "" {
		sysDrive := os.Getenv("SystemDrive")
		if sysDrive == "" {
			sysDrive = "C:"
		}
		cands = append(cands, filepath.Join(sysDrive+string(os.PathSeparator), "Users", user, "AppData", "Local", "AntigravityManager", "profiles_registry.json"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cands = append(cands, filepath.Join(home, "AppData", "Local", "AntigravityManager", "profiles_registry.json"))
	}
	if realHome := GetRealUserHomeWindows(); realHome != "" {
		cands = append(cands, filepath.Join(realHome, "AppData", "Local", "AntigravityManager", "profiles_registry.json"))
	}
	return cands
}

// IsAgyPatched checks if the given agy binary has the patched eligibility gate pattern
func IsAgyPatched(binPath string) bool {
	if binPath == "" {
		return false
	}
	fi, err := os.Stat(binPath)
	if err != nil || fi.IsDir() {
		return false
	}
	cacheKey := strings.ToLower(filepath.Clean(binPath))
	sz := fi.Size()
	mt := fi.ModTime().UnixNano()

	agyPatchCacheMu.Lock()
	if cached, ok := agyPatchCache[cacheKey]; ok && cached.size == sz && cached.modNano == mt {
		agyPatchCacheMu.Unlock()
		return cached.patched
	}
	agyPatchCacheMu.Unlock()

	data, err := os.ReadFile(binPath)
	if err != nil {
		return false
	}
	patched := false
	// Pattern check: 48 85 c0 90 0f 85
	patchedSeq := []byte{0x48, 0x85, 0xc0, 0x90, 0x0f, 0x85}
	if bytes.Contains(data, patchedSeq) {
		patched = true
	} else {
		// Also check cliGateX64, cliGateARM64
		for _, g := range []BinaryGate{cliGateX64, cliGateARM64, cliGateARM64_W4, cliGateARM64_W2, cliGateAuthX64, cliGateAuthARM64} {
			if g.Patched != nil && g.Patched.Find(data) != -1 {
				patched = true
				break
			}
		}
	}

	agyPatchCacheMu.Lock()
	agyPatchCache[cacheKey] = agyPatchCacheEntry{size: sz, modNano: mt, patched: patched}
	agyPatchCacheMu.Unlock()
	return patched
}

// FindAllManagedProfileAgyBins returns only the agy binary paths inside managed profiles
// without reading multi-hundred-MB binaries to check patch state.
func FindAllManagedProfileAgyBins() []string {
	profs := findManagedProfilesInternal(false)
	var bins []string
	for _, p := range profs {
		if p.AgyBin != "" {
			bins = append(bins, p.AgyBin)
		}
	}
	return bins
}

// FindAllManagedProfiles discovers all profiles registered in AntigravityManager or C:\Antigravity-Profiles
func FindAllManagedProfiles() []ManagedProfile {
	return findManagedProfilesInternal(true)
}

func findManagedProfilesInternal(checkPatched bool) []ManagedProfile {
	var profiles []ManagedProfile
	seen := make(map[string]bool)

	addProfileDir := func(profDir string) {
		clean := filepath.Clean(profDir)
		key := strings.ToLower(clean)
		if seen[key] {
			return
		}
		if fi, err := os.Stat(clean); err != nil || !fi.IsDir() {
			return
		}
		seen[key] = true

		baseName := filepath.Base(clean)
		prof := ManagedProfile{
			Name: baseName,
			Path: clean,
		}

		// Look for agy binary inside this profile
		cands := []string{
			filepath.Join(clean, "home", ".gemini", "bin", "agy.exe"),
			filepath.Join(clean, "home", ".gemini", "bin", "agy"),
			filepath.Join(clean, ".gemini", "bin", "agy.exe"),
			filepath.Join(clean, ".gemini", "bin", "agy"),
			filepath.Join(clean, "bin", "agy.exe"),
			filepath.Join(clean, "bin", "agy"),
		}
		for _, c := range cands {
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Size() > 0 {
				prof.AgyBin = c
				if checkPatched {
					prof.IsPatched = IsAgyPatched(c)
				}
				break
			}
		}

		profiles = append(profiles, prof)
	}

	// 1. Read profiles_registry.json candidates
	for _, regFile := range findManagerRegistryCandidates() {
		data, err := os.ReadFile(regFile)
		if err != nil {
			continue
		}
		var paths []string
		if json.Unmarshal(data, &paths) == nil {
			for _, p := range paths {
				if p != "" {
					addProfileDir(p)
				}
			}
		}
	}

	// 2. Scan standard profile directories
	for _, root := range []string{`C:\Antigravity-Profiles`, `D:\Antigravity-Profiles`} {
		if entries, err := os.ReadDir(root); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					addProfileDir(filepath.Join(root, e.Name()))
				}
			}
		}
	}

	return profiles
}

