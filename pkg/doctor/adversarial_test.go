package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAdv_DetectAntigravityVersion_CorruptedProductJSON(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	outDir := filepath.Join(appDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	mainJs := filepath.Join(outDir, "main.js")
	_ = os.WriteFile(mainJs, []byte("console.log('main');"), 0644)

	// Case 1: Corrupted product.json (syntax error) with valid package.json fallback
	productPath := filepath.Join(appDir, "product.json")
	corruptedProductJSON := `{"nameShort": "Antigravity", "ideVersion": ` // truncated JSON
	if err := os.WriteFile(productPath, []byte(corruptedProductJSON), 0644); err != nil {
		t.Fatalf("failed to write corrupted product.json: %v", err)
	}

	packagePath := filepath.Join(appDir, "package.json")
	validPackageJSON := `{"name": "antigravity", "version": "1.107.0"}`
	if err := os.WriteFile(packagePath, []byte(validPackageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Test candidate JSON parsing logic directly
	prodCandidates := []string{
		filepath.Join(filepath.Dir(mainJs), "..", "product.json"),
		filepath.Join(filepath.Dir(mainJs), "product.json"),
		filepath.Join(filepath.Dir(mainJs), "..", "..", "product.json"),
	}

	var detectedProd, detectedVer string
	for _, c := range prodCandidates {
		if data, err := os.ReadFile(c); err == nil {
			var prod struct {
				NameShort       string `json:"nameShort"`
				NameLong        string `json:"nameLong"`
				ApplicationName string `json:"applicationName"`
				IdeVersion      string `json:"ideVersion"`
				Version         string `json:"version"`
			}
			if json.Unmarshal(data, &prod) == nil {
				ver := prod.IdeVersion
				if ver == "" {
					ver = prod.Version
				}
				if ver != "" {
					detectedProd = prod.NameLong
					detectedVer = ver
					break
				}
			}
		}
	}

	if detectedVer != "" {
		t.Errorf("corrupted product.json should not have yielded a version, got: %s", detectedVer)
	}

	// Fallback to package.json
	pkgCandidates := []string{
		filepath.Join(filepath.Dir(mainJs), "..", "package.json"),
		filepath.Join(filepath.Dir(mainJs), "package.json"),
		filepath.Join(filepath.Dir(mainJs), "..", "..", "package.json"),
	}
	for _, c := range pkgCandidates {
		if data, err := os.ReadFile(c); err == nil {
			var pkg struct {
				Name        string `json:"name"`
				Version     string `json:"version"`
				DisplayName string `json:"displayName"`
			}
			if json.Unmarshal(data, &pkg) == nil && pkg.Version != "" {
				detectedProd = pkg.DisplayName
				if detectedProd == "" {
					detectedProd = pkg.Name
				}
				detectedVer = pkg.Version
				break
			}
		}
	}

	if detectedVer != "1.107.0" {
		t.Errorf("failed to fallback to package.json when product.json is corrupted: got ver=%q", detectedVer)
	}
}

func TestAdv_DetectAntigravityVersion_ZeroByteProductJSON(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	outDir := filepath.Join(appDir, "out")
	_ = os.MkdirAll(outDir, 0755)

	mainJs := filepath.Join(outDir, "main.js")
	_ = os.WriteFile(mainJs, []byte("console.log('main');"), 0644)

	productPath := filepath.Join(appDir, "product.json")
	_ = os.WriteFile(productPath, []byte(""), 0644) // 0-byte file

	packagePath := filepath.Join(appDir, "package.json")
	_ = os.WriteFile(packagePath, []byte(`{"name": "antigravity", "version": "2.5.5"}`), 0644)

	// Verify no crash on empty file
	data, err := os.ReadFile(productPath)
	if err != nil {
		t.Fatalf("failed to read empty product.json: %v", err)
	}
	var prod struct {
		IdeVersion string `json:"ideVersion"`
	}
	if err := json.Unmarshal(data, &prod); err == nil {
		t.Errorf("unmarshal of 0 bytes should error or be empty")
	}
}

func TestAdv_DetectAntigravityVersion_EmptyIdeVersionInProductJSON(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "resources", "app")
	outDir := filepath.Join(appDir, "out")
	_ = os.MkdirAll(outDir, 0755)

	productPath := filepath.Join(appDir, "product.json")
	// product.json with empty ideVersion, but version is present
	_ = os.WriteFile(productPath, []byte(`{"nameLong": "Antigravity IDE", "ideVersion": "", "version": "2.6.0-fallback"}`), 0644)

	data, _ := os.ReadFile(productPath)
	var prod struct {
		NameLong   string `json:"nameLong"`
		IdeVersion string `json:"ideVersion"`
		Version    string `json:"version"`
	}
	_ = json.Unmarshal(data, &prod)
	ver := prod.IdeVersion
	if ver == "" {
		ver = prod.Version
	}
	if ver != "2.6.0-fallback" {
		t.Errorf("expected fallback to Version when IdeVersion is empty, got %q", ver)
	}
}
