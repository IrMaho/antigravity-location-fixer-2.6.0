package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestFindCandidateBinaries_NoDuplicates verifies that FindCandidateBinaries()
// returns strictly unique paths with no duplicates under various normalization schemes.
func TestFindCandidateBinaries_NoDuplicates(t *testing.T) {
	candidates := FindCandidateBinaries()
	if len(candidates) == 0 {
		t.Fatal("expected non-empty candidate list")
	}

	rawSeen := make(map[string]int)
	cleanSeen := make(map[string]int)

	for i, c := range candidates {
		if c == "" {
			t.Errorf("candidate[%d] is empty string", i)
		}
		rawSeen[c]++
		cleanSeen[filepath.Clean(strings.ToLower(c))]++
	}

	for path, count := range rawSeen {
		if count > 1 {
			t.Errorf("RAW DUPLICATE found (%d occurrences): %q", count, path)
		}
	}

	for path, count := range cleanSeen {
		if count > 1 {
			t.Errorf("CASE-INSENSITIVE / CLEAN DUPLICATE found (%d occurrences): %q", count, path)
		}
	}
}

// TestFindCandidateBinaries_AbsenceAndEnvironmentStress verifies that
// FindCandidateBinaries never panics or deadlocks when environment variables
// (USERPROFILE, LOCALAPPDATA, PROGRAMFILES, HOME) are empty, malformed, or missing.
func TestFindCandidateBinaries_AbsenceAndEnvironmentStress(t *testing.T) {
	envVars := []string{
		"USERPROFILE",
		"LOCALAPPDATA",
		"PROGRAMFILES",
		"ProgramFiles(x86)",
		"HOME",
	}

	// Backup existing env
	backup := make(map[string]string)
	for _, env := range envVars {
		backup[env] = os.Getenv(env)
	}
	defer func() {
		for k, v := range backup {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	stressCases := []struct {
		name string
		set  map[string]string
	}{
		{
			name: "all_empty",
			set: map[string]string{
				"USERPROFILE":       "",
				"LOCALAPPDATA":      "",
				"PROGRAMFILES":      "",
				"ProgramFiles(x86)": "",
				"HOME":              "",
			},
		},
		{
			name: "special_chars",
			set: map[string]string{
				"USERPROFILE":       `C:\Users\Special !@#$%^&()`,
				"LOCALAPPDATA":      `C:\Users\Special !@#$%^&()\AppData\Local`,
				"PROGRAMFILES":      `C:\Program Files (x86)\Test`,
				"ProgramFiles(x86)": `C:\Program Files (x86)`,
				"HOME":              `/home/special !@#$%^&*()`,
			},
		},
		{
			name: "trailing_slashes",
			set: map[string]string{
				"USERPROFILE":       `C:\Users\Trailing\`,
				"LOCALAPPDATA":      `C:\Users\Trailing\AppData\Local\\`,
				"PROGRAMFILES":      `C:\Program Files\\\`,
				"ProgramFiles(x86)": `C:\Program Files (x86)\`,
				"HOME":              `/home/trailing///`,
			},
		},
	}

	for _, sc := range stressCases {
		t.Run(sc.name, func(t *testing.T) {
			for k, v := range sc.set {
				os.Setenv(k, v)
			}

			done := make(chan bool, 1)
			var candidates []string
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC in FindCandidateBinaries(): %v", r)
					}
					done <- true
				}()
				candidates = FindCandidateBinaries()
			}()

			select {
			case <-done:
				if len(candidates) == 0 {
					t.Errorf("expected candidates even under empty/unusual env")
				}
				// Verify uniqueness under this env
				seen := make(map[string]int)
				for _, c := range candidates {
					seen[c]++
					if seen[c] > 1 {
						t.Errorf("duplicate found under %s: %s", sc.name, c)
					}
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("DEADLOCK detected in FindCandidateBinaries() under %s", sc.name)
			}
		})
	}
}

// TestFindCandidateBinaries_Concurrency tests race conditions / thread safety.
func TestFindCandidateBinaries_Concurrency(t *testing.T) {
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			res := FindCandidateBinaries()
			if len(res) == 0 {
				t.Errorf("goroutine %d got empty candidates", id)
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded without deadlock or race
	case <-time.After(5 * time.Second):
		t.Fatal("DEADLOCK in concurrent FindCandidateBinaries() calls")
	}
}

func TestSafety_IsAntigravityRunningAndKill(t *testing.T) {
	running := IsAntigravityRunning()
	t.Logf("IsAntigravityRunning() = %v", running)

	count, err := KillAntigravityProcesses()
	t.Logf("KillAntigravityProcesses() = %d, err = %v", count, err)
}

