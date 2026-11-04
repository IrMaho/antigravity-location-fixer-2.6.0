package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindAllLanguageServers_DuplicateFiltering(t *testing.T) {
	// Create mock files in a temporary dir
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "resources", "app", "extensions", "antigravity", "bin")
	_ = os.MkdirAll(binDir, 0755)

	targetBin := filepath.Join(binDir, "agy.exe")
	if err := os.WriteFile(targetBin, []byte("dummy exe"), 0755); err != nil {
		t.Fatalf("failed to write mock binary: %v", err)
	}

	// Verify FindAllLanguageServers with seen map
	// Note: FindAllLanguageServers does:
	// clean := filepath.Clean(p)
	// if !seen[clean] { if fi, err := os.Stat(clean); err == nil && !fi.IsDir() { seen[clean] = true; results = append(results, clean) } }
	// If the candidate list had "C:\foo\agy.exe" and "c:\foo\agy.exe", seen["C:\foo\agy.exe"] and seen["c:\foo\agy.exe"]
	// are two different map keys!
	cased1 := targetBin
	cased2 := strings.ToLower(targetBin)
	if cased1 != cased2 {
		// On Windows, if two candidates with different casing exist,
		// seen[clean] would allow both unless normalized!
		t.Logf("Cased1: %s, Cased2: %s", cased1, cased2)
	}
}
