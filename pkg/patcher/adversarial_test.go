package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =========================================================================
// 1. IDEMPOTENCY TESTS ON UNPACKED RESOURCES
// =========================================================================

func TestAdv_Idempotency_PatchAsarZeroLog_Unpacked(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	outDir := filepath.Join(appDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create fake app out dir: %v", err)
	}

	mainJs := filepath.Join(outDir, "main.js")
	initialContent := "resetIsTierGCPTos(),this._productService.isGoogleInternal\nconsole.log('init');"
	if err := os.WriteFile(mainJs, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write main.js: %v", err)
	}

	// Run 1: Initial patch
	res1, err1 := PatchAsarZeroLog(appDir)
	if err1 != nil {
		t.Fatalf("Run 1 failed: %v", err1)
	}
	t.Logf("Run 1 output: %s", res1)

	contentAfterRun1, err := os.ReadFile(mainJs)
	if err != nil {
		t.Fatalf("failed to read main.js after run 1: %v", err)
	}
	if !strings.Contains(string(contentAfterRun1), "resetIsTierGCPTos(),true") {
		t.Fatalf("Run 1 failed to patch isGoogleInternal")
	}

	// Run 2: Second call (Idempotency)
	res2, err2 := PatchAsarZeroLog(appDir)
	if err2 != nil {
		t.Fatalf("Run 2 failed on already patched unpacked resources: %v", err2)
	}
	t.Logf("Run 2 output: %s", res2)

	contentAfterRun2, err := os.ReadFile(mainJs)
	if err != nil {
		t.Fatalf("failed to read main.js after run 2: %v", err)
	}
	if string(contentAfterRun1) != string(contentAfterRun2) {
		t.Errorf("Idempotency violation in PatchAsarZeroLog: content changed between Run 1 and Run 2!\nRun1: %q\nRun2: %q",
			string(contentAfterRun1), string(contentAfterRun2))
	}

	// Run 3: Third call
	res3, err3 := PatchAsarZeroLog(appDir)
	if err3 != nil {
		t.Fatalf("Run 3 failed: %v", err3)
	}
	t.Logf("Run 3 output: %s", res3)

	contentAfterRun3, err := os.ReadFile(mainJs)
	if err != nil {
		t.Fatalf("failed to read main.js after run 3: %v", err)
	}
	if string(contentAfterRun1) != string(contentAfterRun3) {
		t.Errorf("Idempotency violation in PatchAsarZeroLog: content changed between Run 1 and Run 3")
	}
}

func TestAdv_Idempotency_PatchAsarAntiUpdate_Unpacked(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create fake app dir: %v", err)
	}

	ymlPath := filepath.Join(appDir, "app-update.yml")
	initialYml := "provider: generic\nurl: https://update.antigravity.google.com/\nupdaterCacheDirName: antigravity-updater\n"
	if err := os.WriteFile(ymlPath, []byte(initialYml), 0644); err != nil {
		t.Fatalf("failed to write app-update.yml: %v", err)
	}

	// Run 1: Initial patch
	res1, err1 := PatchAsarAntiUpdate(appDir)
	if err1 != nil {
		t.Fatalf("Run 1 failed: %v", err1)
	}
	t.Logf("Run 1 output: %s", res1)

	content1, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("failed to read yml after run 1: %v", err)
	}
	if !strings.Contains(string(content1), "127.0.0.1:0") {
		t.Fatalf("app-update.yml was not neutralized in Run 1")
	}

	// Run 2: Second call (Idempotency)
	res2, err2 := PatchAsarAntiUpdate(appDir)
	if err2 != nil {
		t.Fatalf("Run 2 failed: %v", err2)
	}
	t.Logf("Run 2 output: %s", res2)

	content2, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("failed to read yml after run 2: %v", err)
	}
	if string(content1) != string(content2) {
		t.Errorf("Idempotency violation in PatchAsarAntiUpdate: content changed between Run 1 and Run 2")
	}

	// Run 3: Third call
	res3, err3 := PatchAsarAntiUpdate(appDir)
	if err3 != nil {
		t.Fatalf("Run 3 failed: %v", err3)
	}
	t.Logf("Run 3 output: %s", res3)

	content3, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("failed to read yml after run 3: %v", err)
	}
	if string(content1) != string(content3) {
		t.Errorf("Idempotency violation in PatchAsarAntiUpdate: content changed between Run 1 and Run 3")
	}
}

func TestAdv_Idempotency_PatchVsCodeExtension_MinifiedStandard(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "extension.js")

	sample := `let Pc=pb&&Kd&&a&&!bd;Pc&&g.appendLine("[INSTALL] Release channel change detected");g.appendLine("[INSTALL] Checking Antigravity releases...");`
	if err := os.WriteFile(extPath, []byte(sample), 0644); err != nil {
		t.Fatalf("failed to write fake extension.js: %v", err)
	}

	// Run 1
	res1, err1 := PatchVsCodeExtension(extPath)
	if err1 != nil {
		t.Fatalf("Run 1 failed: %v", err1)
	}
	t.Logf("Run 1 output: %s", res1)

	content1, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension.js after run 1: %v", err)
	}

	// Run 2: Idempotency
	res2, err2 := PatchVsCodeExtension(extPath)
	if err2 != nil {
		t.Fatalf("Run 2 failed: %v", err2)
	}
	t.Logf("Run 2 output: %s", res2)
	if !strings.Contains(res2, "Already patched") {
		t.Errorf("Run 2 did not report 'Already patched': got %s", res2)
	}

	content2, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension.js after run 2: %v", err)
	}
	if string(content1) != string(content2) {
		t.Errorf("Idempotency violation in PatchVsCodeExtension: content modified on second run!")
	}

	// Run 3
	res3, err3 := PatchVsCodeExtension(extPath)
	if err3 != nil {
		t.Fatalf("Run 3 failed: %v", err3)
	}
	t.Logf("Run 3 output: %s", res3)

	content3, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension.js after run 3: %v", err)
	}
	if string(content1) != string(content3) {
		t.Errorf("Idempotency violation in PatchVsCodeExtension: content modified on third run!")
	}

	// Check downloadGuardCode occurrences
	guardCount := strings.Count(string(content3), "Skipping version check and re-download")
	if guardCount != 1 {
		t.Errorf("Download guard injected %d times (expected exactly 1)", guardCount)
	}
}

// =========================================================================
// 2. BOUNDARY CONDITIONS: MISSING & CORRUPTED MAIN.JS
// =========================================================================

func TestAdv_Boundary_MissingMainJs_Unpacked(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Unpacked app dir with NO main.js anywhere
	res, err := PatchAsarZeroLog(appDir)
	if err != nil {
		t.Fatalf("unexpected error on missing main.js in unpacked app: %v", err)
	}
	t.Logf("Missing main.js response: %s", res)
	if !strings.Contains(res, "Unpacked resources/app detected") {
		t.Errorf("expected unpacked resources detection message, got: %s", res)
	}
}

func TestAdv_Boundary_CorruptedMainJs_Unpacked(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	outDir := filepath.Join(appDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	mainJs := filepath.Join(outDir, "main.js")
	corruptedContent := "var a = 123; function foo() { return null; } // no target pattern here"
	if err := os.WriteFile(mainJs, []byte(corruptedContent), 0644); err != nil {
		t.Fatalf("failed to write corrupted main.js: %v", err)
	}

	// PatchAsarZeroLog should not crash or corrupt the file if target pattern is absent
	res, err := PatchAsarZeroLog(appDir)
	if err != nil {
		t.Fatalf("unexpected error when main.js lacks pattern: %v", err)
	}
	t.Logf("Corrupted main.js response: %s", res)

	contentAfter, err := os.ReadFile(mainJs)
	if err != nil {
		t.Fatalf("failed to read main.js: %v", err)
	}
	if string(contentAfter) != corruptedContent {
		t.Errorf("corrupted main.js was unexpectedly modified: %s", string(contentAfter))
	}
}

// =========================================================================
// 3. ADVERSARIAL MINIFIED JS: EDGE-CASE VARIABLE NAMES
// =========================================================================

func TestAdv_Adversarial_MinifiedJs_EdgeCaseVariableNames(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "extension.js")

	// Notice variable name is Xc instead of Pc, and dependencies are different identifiers:
	// In different builds or minor updates, bundler minifies identifiers differently
	sample := `let Xc=qb&&Ld&&b&&!cd;Xc&&g.appendLine("[INSTALL] Release channel change detected");g.appendLine("[INSTALL] Checking Antigravity releases...");`
	if err := os.WriteFile(extPath, []byte(sample), 0644); err != nil {
		t.Fatalf("failed to write extension.js: %v", err)
	}

	// Run 1: Patch
	res1, err1 := PatchVsCodeExtension(extPath)
	if err1 != nil {
		t.Logf("Run 1 error on non-standard minified variable name: %v", err1)
		return
	}
	t.Logf("Run 1 result: %s", res1)

	content1, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension.js: %v", err)
	}

	// Check if guard was injected
	hasGuard := strings.Contains(string(content1), "Skipping version check and re-download")
	t.Logf("Guard injected: %v", hasGuard)

	// Run 2: Idempotency stress test
	res2, err2 := PatchVsCodeExtension(extPath)
	t.Logf("Run 2 result: %s (err: %v)", res2, err2)

	content2, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read extension.js after run 2: %v", err)
	}

	guardCount := strings.Count(string(content2), "Skipping version check and re-download")
	t.Logf("Guard occurrence count after Run 2: %d", guardCount)
	if guardCount > 1 {
		t.Errorf("CRITICAL IDEMPOTENCY BUG DETECTED: Guard code was duplicated %d times on repeated patch when channelChanged identifier was not 'Pc'!", guardCount)
	}
}

// =========================================================================
// 4. PROCESS SAFETY & WHITESPACE HANDLING
// =========================================================================

func TestAdv_ProcessWhitespaceHandling(t *testing.T) {
	// 1. Verify scanWindowsProcess handles target names with spaces and without .exe
	p1 := scanWindowsProcess("Antigravity IDE.exe")
	t.Logf("scanWindowsProcess('Antigravity IDE.exe') = %q", p1)

	p2 := scanWindowsProcess("Antigravity IDE")
	t.Logf("scanWindowsProcess('Antigravity IDE') = %q", p2)

	// 2. Verify IsAntigravityRunning() does not crash or panic
	isRunning := IsAntigravityRunning()
	t.Logf("IsAntigravityRunning() = %v", isRunning)

	// 3. Check FindRunningLanguageServer with nil logger callback
	pLs := FindRunningLanguageServer(nil)
	t.Logf("FindRunningLanguageServer(nil) = %q", pLs)

	// 4. Check FindRunningAntigravityApp with nil logger callback
	pApp := FindRunningAntigravityApp(nil)
	t.Logf("FindRunningAntigravityApp(nil) = %q", pApp)

	// 5. Check FindRunningLanguageServerPID() on Windows
	pid := FindRunningLanguageServerPID()
	t.Logf("FindRunningLanguageServerPID() = %d", pid)
	if pLs != "" && pid == 0 {
		t.Errorf("DEFECT: Active language server was discovered at %q, but FindRunningLanguageServerPID() returned 0 due to missing Windows implementation (calling pgrep on Windows)", pLs)
	}
}
