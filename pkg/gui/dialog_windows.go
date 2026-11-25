//go:build windows

package gui

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modComdlg32           = windows.NewLazySystemDLL("comdlg32.dll")
	procGetOpenFileNameW  = modComdlg32.NewProc("GetOpenFileNameW")
	modShell32            = windows.NewLazySystemDLL("shell32.dll")
	procSHBrowseForFolder = modShell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromID   = modShell32.NewProc("SHGetPathFromIDListW")
	modOle32              = windows.NewLazySystemDLL("ole32.dll")
	procCoInitializeEx    = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize    = modOle32.NewProc("CoUninitialize")
	procCoTaskMemFree     = modOle32.NewProc("CoTaskMemFree")
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

func encodeMultiStringUTF16(parts ...string) []uint16 {
	var out []uint16
	for _, p := range parts {
		u, err := syscall.UTF16FromString(p)
		if err == nil {
			out = append(out, u...)
		}
	}
	out = append(out, 0)
	return out
}

func desktopInitialDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		desk := filepath.Join(home, "Desktop")
		if fi, stErr := os.Stat(desk); stErr == nil && fi.IsDir() {
			return desk
		}
		return home
	}
	return ""
}

func browseFileWindowsNative() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const (
		coinitApartmentThreaded = 0x2
		ofnPathMustExist        = 0x00000800
		ofnFileMustExist        = 0x00001000
		ofnNoChangeDir          = 0x00000008
		ofnExplorer             = 0x00080000
	)

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hr == 0 || hr == 1 {
		defer procCoUninitialize.Call()
	}

	filter := encodeMultiStringUTF16(
		"Antigravity Targets (*.exe;*.lnk;*.js)", "*.exe;*.lnk;*.js",
		"Shortcuts (*.lnk)", "*.lnk",
		"Executables (*.exe)", "*.exe",
		"JavaScript (*.js)", "*.js",
		"All Files (*.*)", "*.*",
	)
	fileBuf := make([]uint16, 4096)
	title, _ := syscall.UTF16PtrFromString("Select Antigravity IDE Executable, Shortcut, or main.js")
	var initDirPtr *uint16
	if initDir := desktopInitialDir(); initDir != "" {
		initDirPtr, _ = syscall.UTF16PtrFromString(initDir)
	}

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		lpstrFilter:     &filter[0],
		nFilterIndex:    1,
		lpstrFile:       &fileBuf[0],
		nMaxFile:        uint32(len(fileBuf)),
		lpstrInitialDir: initDirPtr,
		lpstrTitle:      title,
		flags:           ofnPathMustExist | ofnFileMustExist | ofnNoChangeDir | ofnExplorer,
	}

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return "", nil
	}
	return windows.UTF16ToString(fileBuf), nil
}

func browseFolderWindowsNative() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const (
		coinitApartmentThreaded = 0x2
		bifReturnOnlyFsDirs     = 0x00000001
		bifNewDialogStyle       = 0x00000040
		bifNonewfOlderButton    = 0x00000200
	)

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hr == 0 || hr == 1 {
		defer procCoUninitialize.Call()
	}

	dispBuf := make([]uint16, windows.MAX_PATH)
	title, _ := syscall.UTF16PtrFromString("Select Antigravity Installation Directory")
	bi := browseInfoW{
		pszDisplayName: &dispBuf[0],
		lpszTitle:      title,
		ulFlags:        bifReturnOnlyFsDirs | bifNewDialogStyle | bifNonewfOlderButton,
	}

	pidl, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil
	}
	defer procCoTaskMemFree.Call(pidl)

	pathBuf := make([]uint16, windows.MAX_PATH*4)
	ok, _, _ := procSHGetPathFromID.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ok == 0 {
		return "", nil
	}
	return windows.UTF16ToString(pathBuf), nil
}
