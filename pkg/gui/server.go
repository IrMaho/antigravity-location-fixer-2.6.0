package gui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tawroot/antigravity-cleaner/pkg/account"
	"github.com/tawroot/antigravity-cleaner/pkg/cleaner"
	"github.com/tawroot/antigravity-cleaner/pkg/doctor"
	"github.com/tawroot/antigravity-cleaner/pkg/patcher"
	"github.com/tawroot/antigravity-cleaner/pkg/proxy"
)

func init() {
	patcher.RegisterPostInstallHook(func(logger patcher.LogFunc) {
		_ = account.SyncTokenBridge(logger)
	})
}

//go:embed assets/*
var assetsFS embed.FS

// Server represents the local loopback backend for the GUI.
type Server struct {
	listener net.Listener
	server   *http.Server
	port     int
}

// Response represents a standard JSON response.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DetectResponse represents auto-detection details for the frontend.
type DetectResponse struct {
	App            string `json:"app"`
	Path           string `json:"path"`
	Detected       bool   `json:"detected"`
	Proxy          string `json:"proxy"`
	Mode           string `json:"mode"`
	VpnActive      bool   `json:"vpn_active"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	WrapperActive  bool   `json:"wrapper_active"`
	RunningPID     int    `json:"running_pid"`
	ProcessRunning bool   `json:"process_running"`
}

// NewServer initializes a new GUI loopback server on an ephemeral port.
func NewServer() (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind loopback listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	s := &Server{
		listener: listener,
		port:     port,
	}

	mux := http.NewServeMux()

	// Static assets sub-filesystem
	subFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded assets: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// REST API routes
	mux.HandleFunc("/api/detect", s.handleDetect)
	mux.HandleFunc("/api/patch", s.handlePatch)
	mux.HandleFunc("/api/launch", s.handleLaunch)
	mux.HandleFunc("/api/create-launcher", s.handleCreateLauncher)
	mux.HandleFunc("/api/browse", s.handleBrowse)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/restore", s.handleRestore)
	mux.HandleFunc("/api/doctor", s.handleDoctor)
	mux.HandleFunc("/api/report", s.handleReport)
	mux.HandleFunc("/api/privacy", s.handlePrivacy)
	mux.HandleFunc("/api/open-external", s.handleOpenExternal)
	mux.HandleFunc("/api/exit", s.handleExit)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Port returns the assigned local port.
func (s *Server) Port() int {
	return s.port
}

// URL returns the loopback HTTP URL.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", s.port)
}

// Start runs the server in background.
func (s *Server) Start() error {
	return s.server.Serve(s.listener)
}

// Close terminates the server listener.
func (s *Server) Close() error {
	return s.server.Close()
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// resolveProxyForMode resolves the proxy URL and friendly description based on selected network mode.
// Mode can be: "auto", "direct" / "system_vpn", or "proxy" / "local_proxy".
func resolveProxyForMode(mode string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "direct", "system_vpn", "vpn":
		return "direct://", "Direct / System VPN (Surfshark/WireGuard)"
	case "proxy", "local_proxy":
		active := proxy.DetectActiveProxies()
		if len(active) > 0 {
			for _, p := range active {
				if p.Type == "socks5" || p.Type == "mixed" {
					return p.URL, fmt.Sprintf("%s (%s)", p.URL, p.Provider)
				}
			}
			return active[0].URL, fmt.Sprintf("%s (%s)", active[0].URL, active[0].Provider)
		}
		return "", "No active proxy on 127.0.0.1"
	default: // "auto"
		best, err := proxy.GetBestProxy()
		if err == nil {
			return best.URL, fmt.Sprintf("%s (%s)", best.URL, best.Provider)
		}
		return "direct://", "Direct socket (no proxy active)"
	}
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	customPath := strings.TrimSpace(r.URL.Query().Get("path"))

	var targetPath string
	if customPath != "" {
		if fi, err := os.Stat(customPath); err == nil {
			targetPath = customPath
			if fi.IsDir() {
				candidates := []string{
					filepath.Join(customPath, "Antigravity IDE.exe"),
					filepath.Join(customPath, "Antigravity.exe"),
					filepath.Join(customPath, "resources", "app", "out", "main.js"),
					filepath.Join(customPath, "resources", "app", "main.js"),
					filepath.Join(customPath, "resources", "bin", "language_server.exe"),
				}
				for _, c := range candidates {
					if fileExists(c) {
						targetPath = c
						break
					}
				}
			}
		}
	}

	if targetPath == "" {
		targetPath = patcher.FindIdeMainJs()
	}
	if targetPath == "" {
		targetPath = proxy.FindAntigravityExecutable()
	}
	if targetPath == "" {
		targetPath = patcher.FindLanguageServerModular(nil)
		if targetPath == "" {
			targetPath = patcher.FindLanguageServer()
		}
	}
	if targetPath == "" {
		targetPath = patcher.FindAgy()
	}

	detected := targetPath != ""

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = "auto"
	}
	vpnActive := proxy.HasActiveVpnInterface() || proxy.CheckDirectConnectivity(800*time.Millisecond)
	_, proxyDesc := resolveProxyForMode(mode)

	runningPID := patcher.FindRunningLanguageServerPID()
	wrapperActive := false
	if targetPath != "" {
		dir := filepath.Dir(targetPath)
		wm := patcher.NewWrapperManager(dir, nil)
		wrapperActive = wm.IsWrapperInstalled()
	}

	resp := DetectResponse{
		App:            "Google Antigravity 2.x (IDE & Language Server)",
		Path:           targetPath,
		Detected:       detected,
		Proxy:          proxyDesc,
		Mode:           mode,
		VpnActive:      vpnActive,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		WrapperActive:  wrapperActive,
		RunningPID:     runningPID,
		ProcessRunning: runningPID > 0,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	customPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if customPath == "" || mode == "" {
		var reqBody struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if customPath == "" {
			customPath = strings.TrimSpace(reqBody.Path)
		}
		if mode == "" {
			mode = strings.TrimSpace(reqBody.Mode)
		}
	}
	if mode == "" {
		mode = "auto"
	}

	proxyURL, _ := resolveProxyForMode(mode)

	_, _ = patcher.KillAntigravityProcesses()

	var customMainJs, customLs, customExt, customAsar string
	if customPath != "" {
		if fi, err := os.Stat(customPath); err == nil {
			if fi.IsDir() {
				for _, sub := range []string{
					filepath.Join("resources", "app", "out", "main.js"),
					filepath.Join("resources", "app", "main.js"),
				} {
					p := filepath.Join(customPath, sub)
					if fileExists(p) {
						customMainJs = p
						break
					}
				}
				customExt = filepath.Join(customPath, "resources", "app", "extensions", "agy", "vscode", "extension.js")
				customAsar = filepath.Join(customPath, "resources", "app.asar")
				customLs = filepath.Join(customPath, "resources", "bin", "language_server.exe")
			} else {
				base := strings.ToLower(filepath.Base(customPath))
				if strings.HasSuffix(base, ".exe") {
					appDir := filepath.Dir(customPath)
					for _, sub := range []string{
						filepath.Join("resources", "app", "out", "main.js"),
						filepath.Join("resources", "app", "main.js"),
					} {
						p := filepath.Join(appDir, sub)
						if fileExists(p) {
							customMainJs = p
							break
						}
					}
					customExt = filepath.Join(appDir, "resources", "app", "extensions", "agy", "vscode", "extension.js")
					customAsar = filepath.Join(appDir, "resources", "app.asar")
					customLs = filepath.Join(appDir, "resources", "bin", "language_server.exe")
				} else if strings.HasSuffix(base, ".js") {
					customMainJs = customPath
				} else if strings.HasSuffix(base, ".asar") {
					customAsar = customPath
				}
			}
		}
	}

	var patchedDetails []string
	var errors []string

	// Target 1: IDE main.js
	if _, err := patcher.PatchIdeMainJs(customMainJs); err == nil {
		patchedDetails = append(patchedDetails, "IDE main.js")
	} else if customMainJs != "" && fileExists(customMainJs) {
		errors = append(errors, fmt.Sprintf("IDE main.js: %v", err))
	}

	// Target 2: Core Language Server
	if _, err := patcher.PatchLanguageServer(customLs); err == nil {
		patchedDetails = append(patchedDetails, "Language Server")
	} else if customLs != "" && fileExists(customLs) {
		errors = append(errors, fmt.Sprintf("Language Server: %v", err))
	}

	// Target 3: Antigravity CLI (agy)
	if _, err := patcher.PatchAgy(""); err == nil {
		patchedDetails = append(patchedDetails, "CLI (agy)")
	}

	// Target 4: VS Code extension
	if _, err := patcher.PatchVsCodeExtension(customExt); err == nil {
		patchedDetails = append(patchedDetails, "VS Code Extension")
	}

	// Target 5: ASAR Zero-Log
	if _, err := patcher.PatchAsarZeroLog(customAsar); err == nil {
		patchedDetails = append(patchedDetails, "Zero-Log ASAR")
	}

	// Target 6: Clear 429 quota cache & tokens
	_, _ = cleaner.SurgicalReset429("", false)

	// Target 7: Desktop shortcut
	_, _ = proxy.GenerateNoTunDesktopLauncherCustom(customPath, proxyURL)

	// Check if any core target succeeded
	success := len(patchedDetails) > 0
	var msg string
	if success {
		msg = fmt.Sprintf("All targets unlocked: [%s] & 429 Quota Reset!", strings.Join(patchedDetails, ", "))
	} else {
		if len(errors) > 0 {
			msg = "Error patching targets: " + strings.Join(errors, "; ")
		} else {
			msg = "No Antigravity targets found! Please use [📁 Browse...] or [📂 Folder] to select your installation path."
		}
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: msg,
	})
}

func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	customPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if customPath == "" || mode == "" {
		var reqBody struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if customPath == "" {
			customPath = strings.TrimSpace(reqBody.Path)
		}
		if mode == "" {
			mode = strings.TrimSpace(reqBody.Mode)
		}
	}
	if mode == "" {
		mode = "auto"
	}

	proxyURL, _ := resolveProxyForMode(mode)
	if proxyURL == "" && strings.ToLower(mode) == "proxy" {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "ERROR: no active proxy detected on 127.0.0.1 (please start your proxy client)",
		})
		return
	}

	err := proxy.LaunchAntigravityWithProxy(customPath, proxyURL, nil)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Antigravity launched successfully with injected proxy!",
	})
}

func (s *Server) handleCreateLauncher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	customPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if customPath == "" || mode == "" {
		var reqBody struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if customPath == "" {
			customPath = strings.TrimSpace(reqBody.Path)
		}
		if mode == "" {
			mode = strings.TrimSpace(reqBody.Mode)
		}
	}
	if mode == "" {
		mode = "auto"
	}

	proxyURL, _ := resolveProxyForMode(mode)
	path, err := proxy.GenerateNoTunDesktopLauncherCustom(customPath, proxyURL)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Error creating launcher: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Desktop shortcut created at " + path,
	})
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	browseType := r.URL.Query().Get("type") // "folder" or "file"

	var selectedPath string
	var err error

	if runtime.GOOS == "windows" {
		if browseType == "folder" {
			selectedPath, err = browseFolderWindows()
		} else {
			selectedPath, err = browseFileWindows()
		}
	} else if runtime.GOOS == "darwin" {
		selectedPath, err = browseMac(browseType)
	} else {
		selectedPath, err = browseLinux(browseType)
	}

	if err != nil || strings.TrimSpace(selectedPath) == "" {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "No file or folder selected",
		})
		return
	}

	selectedPath = strings.TrimSpace(selectedPath)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
	}{
		Success: true,
		Path:    selectedPath,
	})
}

func browseFileWindows() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	script := `
[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
$d = New-Object System.Windows.Forms.OpenFileDialog
$d.Title = 'Select Antigravity IDE Executable or main.js'
$d.Filter = 'Antigravity Targets (*.exe;*.js)|*.exe;*.js|Executables (*.exe)|*.exe|JavaScript (*.js)|*.js|All Files (*.*)|*.*'
$desktop = [Environment]::GetFolderPath('Desktop')
$d.InitialDirectory = $desktop
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    Write-Output $d.FileName
}
`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func browseFolderWindows() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	script := `
[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = 'Select Antigravity Installation Directory (e.g. on Desktop)'
$desktop = [Environment]::GetFolderPath('Desktop')
$d.SelectedPath = $desktop
$d.ShowNewFolderButton = $false
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    Write-Output $d.SelectedPath
}
`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func browseMac(browseType string) (string, error) {
	prompt := "Select Antigravity File:"
	cmdStr := `POSIX path of (choose file with prompt "` + prompt + `")`
	if browseType == "folder" {
		cmdStr = `POSIX path of (choose folder with prompt "Select Antigravity Folder:")`
	}
	cmd := exec.Command("osascript", "-e", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func browseLinux(browseType string) (string, error) {
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=Select Antigravity Target"}
		if browseType == "folder" {
			args = append(args, "--directory")
		}
		out, err := exec.Command("zenity", args...).Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		arg := "--getopenfilename"
		if browseType == "folder" {
			arg = "--getexistingdirectory"
		}
		out, err := exec.Command("kdialog", arg).Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("no GUI file dialog utility found")
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _ = patcher.KillAntigravityProcesses()

	removed, err := cleaner.SurgicalReset429("", false)
	if err != nil {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: fmt.Sprintf("Cleaned %d cache & session artifacts", len(removed)),
	})
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _ = patcher.KillAntigravityProcesses()

	restoredCount := 0
	if lsPath := patcher.FindLanguageServer(); lsPath != "" {
		if err := patcher.RestoreFile(lsPath); err == nil {
			restoredCount++
		}
	}
	if agyPath := patcher.FindAgy(); agyPath != "" {
		if err := patcher.RestoreFile(agyPath); err == nil {
			restoredCount++
		}
	}
	if idePath := patcher.FindIdeMainJs(); idePath != "" {
		if err := patcher.RestoreFile(idePath); err == nil {
			restoredCount++
		}
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: fmt.Sprintf("Restored %d files from clean .agybak backups", restoredCount),
	})
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	results := doctor.RunDiagnostics(false)
	allOk := true
	for _, res := range results {
		if res.Status == "FAIL" {
			allOk = false
			break
		}
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: allOk,
		Message: "Diagnostics completed",
	})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	report := doctor.GenerateDiagnosticReport("5.4.6")
	doctor.CopyToClipboard(report)
	filePath, _ := doctor.SaveAndOpenReport(report)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  report,
		"filePath": filePath,
	})
}

func (s *Server) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetPath := patcher.FindLanguageServerModular(nil)
	if targetPath == "" {
		targetPath = patcher.FindLanguageServer()
	}
	if targetPath == "" {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Language server binary not found",
		})
		return
	}

	binDir := filepath.Dir(targetPath)
	wm := patcher.NewWrapperManager(binDir, nil)

	if wm.IsWrapperInstalled() {
		if err := wm.RemoveWrapper(); err != nil {
			_ = json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "Failed to remove wrapper: " + err.Error(),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(Response{
			Success: true,
			Message: "Zero-Telemetry wrapper disabled. Original binary restored.",
		})
		return
	}

	opts := patcher.DefaultEnterpriseFlagsOptions()
	if err := wm.InstallWrapper(opts); err != nil {
		_ = json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to install wrapper: " + err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Zero-Telemetry & Enterprise wrapper enabled (7 privacy flags injected)!",
	})
}

func (s *Server) handleOpenExternal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	target := r.URL.Query().Get("url")
	if target == "" {
		target = "https://t.me/dallrep"
	}
	if !strings.HasPrefix(target, "https://t.me/") && !strings.HasPrefix(target, "https://github.com/") {
		http.Error(w, "Forbidden URL", http.StatusForbidden)
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	_ = cmd.Start()

	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Opening external link: " + target,
	})
}

func (s *Server) handleExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Exiting",
	})
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}
