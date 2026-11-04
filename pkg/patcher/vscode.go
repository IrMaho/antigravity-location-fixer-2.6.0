package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	// channelChangedRe matches both unminified and minified release channel change assignments
	channelChangedRe  = regexp.MustCompile(`(?:const\s+isChannelChanged\s*=\s*manifestFetched\s*&&\s*lastInstalledUrl\s*!==\s*releaseBaseUrl;|let\s+Pc\s*=\s*pb\s*&&\s*Kd\s*&&\s*a\s*&&\s*!bd;)`)
	channelPatched    = `const isChannelChanged = false;`
	channelPatchedMin = `let Pc = false;`

	// downloadGuardTarget matches unminified and minified outputChannel.appendLine logging lines
	downloadGuardTarget = regexp.MustCompile(`((?:outputChannel|[a-zA-Z0-9_$]+)\.appendLine\(\s*['"]\[INSTALL\]\s*Checking Antigravity releases\.\.\.['"]\s*\);)`)
	downloadGuardCode   = `{const __fs=typeof require==='function'?require('fs'):null;const __path=typeof require==='function'?require('path'):null;const __logger=(typeof outputChannel!=='undefined'?outputChannel:(typeof g!=='undefined'?g:null));const __primary=(typeof options!=='undefined'&&options.targetPathOverride)||(typeof getInstalledTargetPath==='function'&&getInstalledTargetPath())||(typeof B!=='undefined'?B:(typeof Ub==='function'&&Ub()));if(__fs&&__path&&__primary){const __candidates=[__primary,__path.join(__path.dirname(__primary),'antigravity'+(__path.extname(__primary)||''))];for(const __p of __candidates){if(__p&&__fs.existsSync(__p)){if(__logger&&__logger.appendLine){__logger.appendLine('[INSTALL] Existing binary found at '+__p+'. Skipping version check and re-download.');}return __p;}}}}`
)

// FindVsCodeExtension finds the extension.js of google.google-antigravity
func FindVsCodeExtension() string {
	home, _ := os.UserHomeDir()
	candidateDirs := []string{
		filepath.Join(home, ".vscode", "extensions"),
		filepath.Join(home, ".vscode-insiders", "extensions"),
		filepath.Join(home, ".vscode-oss", "extensions"),
		filepath.Join(home, ".vscode-server", "extensions"),
		filepath.Join(home, ".antigravity-ide", "extensions"),
		filepath.Join(home, ".antigravity", "extensions"),
	}

	if runtime.GOOS == "windows" {
		localApp := os.Getenv("LOCALAPPDATA")
		candidateDirs = append(candidateDirs,
			filepath.Join(localApp, "Programs", "Antigravity IDE", "resources", "app", "extensions"),
			filepath.Join(home, "AppData", "Local", "Programs", "Antigravity IDE", "resources", "app", "extensions"),
			filepath.Join(localApp, "Programs", "Antigravity", "resources", "app", "extensions"),
			filepath.Join(home, "AppData", "Local", "Programs", "Antigravity", "resources", "app", "extensions"),
			filepath.Join(localApp, "Programs", "antigravity", "resources", "app", "extensions"),
			filepath.Join(home, "AppData", "Local", "Programs", "antigravity", "resources", "app", "extensions"),
		)
	} else if runtime.GOOS == "linux" {
		candidateDirs = append(candidateDirs,
			"/usr/share/antigravity/resources/app/extensions",
			"/usr/share/antigravity-ide/resources/app/extensions",
			"/opt/antigravity/resources/app/extensions",
			"/opt/Antigravity IDE/resources/app/extensions",
		)
	}

	var latestExtJs string
	var latestModTime int64

	for _, dir := range candidateDirs {
		// 1. Direct subfolder check for 'agy' extension in bundled builds
		directAgy := filepath.Join(dir, "agy")
		if fi, err := os.Stat(directAgy); err == nil && fi.IsDir() {
			for _, sub := range []string{filepath.Join("vscode", "extension.js"), filepath.Join("dist", "extension.js"), "extension.js"} {
				target := filepath.Join(directAgy, sub)
				if fiTarget, errTarget := os.Stat(target); errTarget == nil && !fiTarget.IsDir() {
					if fiTarget.ModTime().Unix() > latestModTime || strings.HasSuffix(target, filepath.Join("vscode", "extension.js")) {
						latestModTime = fiTarget.ModTime().Unix()
						latestExtJs = target
						if strings.HasSuffix(target, filepath.Join("vscode", "extension.js")) {
							break // Prioritize vscode/extension.js for v2.6.0+
						}
					}
				}
			}
		}

		// 2. Scan directory entries for google.google-antigravity-* or agy-*
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && (strings.HasPrefix(e.Name(), "google.google-antigravity-") || strings.EqualFold(e.Name(), "agy") || strings.HasPrefix(e.Name(), "agy-")) {
				for _, sub := range []string{filepath.Join("vscode", "extension.js"), "extension.js", filepath.Join("dist", "extension.js")} {
					extJs := filepath.Join(dir, e.Name(), sub)
					if fi, err := os.Stat(extJs); err == nil && !fi.IsDir() {
						if fi.ModTime().Unix() > latestModTime || strings.HasSuffix(extJs, filepath.Join("vscode", "extension.js")) {
							latestModTime = fi.ModTime().Unix()
							latestExtJs = extJs
							if strings.HasSuffix(extJs, filepath.Join("vscode", "extension.js")) {
								break
							}
						}
					}
				}
			}
		}
	}
	return latestExtJs
}

// PatchVsCodeExtension patches extension.js to disable re-download and lock channel
func PatchVsCodeExtension(extPath string) (string, error) {
	if extPath == "" {
		extPath = FindVsCodeExtension()
		if extPath == "" {
			return "", fmt.Errorf("VS Code extension google.google-antigravity not found")
		}
	}

	data, err := os.ReadFile(extPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", extPath, err)
	}

	content := string(data)
	isAlreadyPatched := (strings.Contains(content, channelPatched) || strings.Contains(content, channelPatchedMin) || strings.Contains(content, "let Pc=false")) &&
		strings.Contains(content, "Skipping version check and re-download")
	if isAlreadyPatched {
		return fmt.Sprintf("Already patched: %s", filepath.Base(filepath.Dir(extPath))), nil
	}

	// Apply Part 1: Channel lock (supports unminified and minified)
	patched := channelChangedRe.ReplaceAllStringFunc(content, func(m string) string {
		if strings.Contains(m, "Pc") {
			return channelPatchedMin
		}
		return channelPatched
	})
	channelApplied := patched != content || strings.Contains(content, channelPatched) || strings.Contains(content, channelPatchedMin)

	// Apply Part 2: Re-download guard injection (supports unminified and minified)
	guardApplied := strings.Contains(patched, "Skipping version check and re-download")
	if downloadGuardTarget.MatchString(patched) {
		patched = downloadGuardTarget.ReplaceAllString(patched, "$1\n"+downloadGuardCode)
		guardApplied = true
	}

	if !channelApplied && !guardApplied {
		return "", fmt.Errorf("patch signatures not found in %s (unsupported or already modified build) — extension was NOT patched", filepath.Base(filepath.Dir(extPath)))
	}

	if _, err := BackupFile(extPath); err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	if err := AtomicWrite(extPath, []byte(patched)); err != nil {
		return "", fmt.Errorf("failed to write patched extension.js: %w", err)
	}

	// Also patch ~/.gemini/bin/antigravity or agy.exe if it exists
	home, _ := os.UserHomeDir()
	downloadedBin := filepath.Join(home, ".gemini", "bin", "antigravity")
	if fi, err := os.Stat(downloadedBin); err == nil && !fi.IsDir() {
		_, _ = PatchAgy(downloadedBin)
	}
	downloadedAgyExe := filepath.Join(home, ".gemini", "bin", "agy.exe")
	if fi, err := os.Stat(downloadedAgyExe); err == nil && !fi.IsDir() {
		_, _ = PatchAgy(downloadedAgyExe)
	}

	return fmt.Sprintf("Successfully patched VS Code extension %s (re-download guard + channel lock)", filepath.Base(filepath.Dir(extPath))), nil
}
