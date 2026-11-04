package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchVsCodeExtension_Unminified(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "extension.js")

	sample := `
// Extension unminified sample
const isChannelChanged = manifestFetched && lastInstalledUrl !== releaseBaseUrl;
outputChannel.appendLine("[INSTALL] Checking Antigravity releases...");
`
	if err := os.WriteFile(extPath, []byte(sample), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	res, err := PatchVsCodeExtension(extPath)
	if err != nil {
		t.Fatalf("PatchVsCodeExtension failed: %v", err)
	}
	if !strings.Contains(res, "Successfully patched") {
		t.Fatalf("expected success message, got: %s", res)
	}

	data, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read patched file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "const isChannelChanged = false;") {
		t.Errorf("channel lock missing in unminified content")
	}
	if !strings.Contains(content, "Skipping version check and re-download") {
		t.Errorf("download guard code missing in unminified content")
	}

	// Verify idempotency
	res2, err2 := PatchVsCodeExtension(extPath)
	if err2 != nil {
		t.Fatalf("idempotent patch failed: %v", err2)
	}
	if !strings.Contains(res2, "Already patched") {
		t.Errorf("expected already patched message, got: %s", res2)
	}
}

func TestPatchVsCodeExtension_Minified(t *testing.T) {
	tmpDir := t.TempDir()
	extPath := filepath.Join(tmpDir, "extension.js")

	sample := `let Pc=pb&&Kd&&a&&!bd;Pc&&g.appendLine("[INSTALL] Release channel change detected");g.appendLine("[INSTALL] Checking Antigravity releases...");`
	if err := os.WriteFile(extPath, []byte(sample), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	res, err := PatchVsCodeExtension(extPath)
	if err != nil {
		t.Fatalf("PatchVsCodeExtension minified failed: %v", err)
	}
	if !strings.Contains(res, "Successfully patched") {
		t.Fatalf("expected success message, got: %s", res)
	}

	data, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("failed to read patched file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "let Pc = false;") {
		t.Errorf("minified channel lock missing")
	}
	if !strings.Contains(content, "Skipping version check and re-download") {
		t.Errorf("minified download guard missing")
	}

	// Verify idempotency
	res2, err2 := PatchVsCodeExtension(extPath)
	if err2 != nil {
		t.Fatalf("idempotent patch failed: %v", err2)
	}
	if !strings.Contains(res2, "Already patched") {
		t.Errorf("expected already patched message, got: %s", res2)
	}
}
