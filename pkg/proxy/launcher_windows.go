//go:build windows

package proxy

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modOle32Link         = windows.NewLazySystemDLL("ole32.dll")
	procCoInitializeLink = modOle32Link.NewProc("CoInitializeEx")
	procCoUninitLink     = modOle32Link.NewProc("CoUninitialize")
	procCoCreateInstance = modOle32Link.NewProc("CoCreateInstance")

	clsidShellLink = windows.GUID{
		Data1: 0x00021401,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	iidIShellLinkW = windows.GUID{
		Data1: 0x000214F9,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	iidIPersistFile = windows.GUID{
		Data1: 0x0000010B,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	winEnvVarPattern = regexp.MustCompile(`%([^%]+)%`)

	regTargetsMu    sync.Mutex
	regTargetsCache []AppTarget
	regTargetsAt    time.Time
)

// FindAntigravityExecutablesFromRegistry scans the Windows Registry uninstall keys
// for official Google Antigravity and Antigravity IDE installations.
func FindAntigravityExecutablesFromRegistry() []AppTarget {
	regTargetsMu.Lock()
	if time.Since(regTargetsAt) < 30*time.Second && regTargetsCache != nil {
		out := make([]AppTarget, len(regTargetsCache))
		copy(out, regTargetsCache)
		regTargetsMu.Unlock()
		return out
	}
	regTargetsMu.Unlock()

	targets := scanAntigravityExecutablesFromRegistryUncached()

	regTargetsMu.Lock()
	regTargetsCache = make([]AppTarget, len(targets))
	copy(regTargetsCache, targets)
	regTargetsAt = time.Now()
	regTargetsMu.Unlock()

	return targets
}

func scanAntigravityExecutablesFromRegistryUncached() []AppTarget {
	var targets []AppTarget
	seen := make(map[string]bool)

	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, k := range keys {
		key, err := registry.OpenKey(k.root, k.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}

		for _, sub := range subkeys {
			sk, err := registry.OpenKey(k.root, k.path+`\`+sub, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			disp, _, _ := sk.GetStringValue("DisplayName")
			loc, _, _ := sk.GetStringValue("InstallLocation")
			icon, _, _ := sk.GetStringValue("DisplayIcon")
			sk.Close()

			dispLower := strings.ToLower(disp)
			if !strings.Contains(dispLower, "antigravity") {
				continue
			}

			isIDE := strings.Contains(dispLower, "ide")

			// 1. Check DisplayIcon
			if icon != "" {
				if idx := strings.Index(icon, ","); idx != -1 {
					icon = icon[:idx]
				}
				icon = strings.Trim(icon, `"`)
				cleanIcon := filepath.Clean(icon)
				iconBase := strings.ToLower(filepath.Base(cleanIcon))
				if iconBase == "antigravity ide.exe" || iconBase == "antigravity.exe" {
					if fi, err := os.Stat(cleanIcon); err == nil && !fi.IsDir() && !seen[cleanIcon] {
						seen[cleanIcon] = true
						name := "Antigravity"
						if isIDE || strings.Contains(strings.ToLower(cleanIcon), "ide") {
							name = "Antigravity IDE"
							isIDE = true
						}
						targets = append(targets, AppTarget{
							Name:    name,
							ExePath: cleanIcon,
							IsIDE:   isIDE,
						})
					}
				}
			}

			// 2. Check InstallLocation
			if loc != "" {
				loc = strings.Trim(loc, `"`)
				for _, exeName := range []string{"Antigravity IDE.exe", "Antigravity.exe", "antigravity.exe"} {
					exe := filepath.Join(loc, exeName)
					cleanExe := filepath.Clean(exe)
					if fi, err := os.Stat(cleanExe); err == nil && !fi.IsDir() && !seen[cleanExe] {
						seen[cleanExe] = true
						name := "Antigravity"
						targetIsIDE := isIDE
						if strings.Contains(strings.ToLower(exeName), "ide") {
							name = "Antigravity IDE"
							targetIsIDE = true
						}
						targets = append(targets, AppTarget{
							Name:    name,
							ExePath: cleanExe,
							IsIDE:   targetIsIDE,
						})
					}
				}
			}
		}
	}

	// 3. Mounted fixed/removable Windows drives (non-C: installs)
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return targets
	}
	username := strings.TrimSpace(os.Getenv("USERNAME"))

	tryAddDriveExe := func(name, exePath string, isIDE bool) {
		cleanExe := filepath.Clean(exePath)
		if !seen[cleanExe] {
			if fi, err := os.Stat(cleanExe); err == nil && !fi.IsDir() {
				seen[cleanExe] = true
				targets = append(targets, AppTarget{
					Name:    name,
					ExePath: cleanExe,
					IsIDE:   isIDE,
				})
			}
		}
	}

	for i := 0; i < 26; i++ {
		if (mask & (1 << uint(i))) == 0 {
			continue
		}
		drv := string(rune('A'+i)) + `:\`
		ptr, err := syscall.UTF16PtrFromString(drv)
		if err != nil {
			continue
		}
		dtype := windows.GetDriveType(ptr)
		if dtype != windows.DRIVE_REMOVABLE && dtype != windows.DRIVE_FIXED {
			continue
		}
		for _, sub := range []string{"", "Programs", "Program Files", "Program Files (x86)", "Apps", "Tools", "Software"} {
			baseDir := drv
			if sub != "" {
				baseDir = filepath.Join(drv, sub)
			}
			tryAddDriveExe("Antigravity IDE", filepath.Join(baseDir, "Antigravity IDE", "Antigravity IDE.exe"), true)
			tryAddDriveExe("Antigravity", filepath.Join(baseDir, "Antigravity", "Antigravity.exe"), false)
			tryAddDriveExe("Antigravity", filepath.Join(baseDir, "antigravity", "Antigravity.exe"), false)
		}
		if username != "" {
			tryAddDriveExe("Antigravity IDE", filepath.Join(drv, "Users", username, "AppData", "Local", "Programs", "Antigravity IDE", "Antigravity IDE.exe"), true)
			tryAddDriveExe("Antigravity", filepath.Join(drv, "Users", username, "AppData", "Local", "Programs", "Antigravity", "Antigravity.exe"), false)
		}
		if entries, err := os.ReadDir(drv); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				low := strings.ToLower(e.Name())
				if strings.HasPrefix(low, "$") || strings.HasPrefix(low, ".") || low == "windows" || low == "system volume information" {
					continue
				}
				topDir := filepath.Join(drv, e.Name())
				if strings.Contains(low, "antigravity") {
					tryAddDriveExe("Antigravity IDE", filepath.Join(topDir, "Antigravity IDE.exe"), true)
					tryAddDriveExe("Antigravity", filepath.Join(topDir, "Antigravity.exe"), false)
				}
				tryAddDriveExe("Antigravity IDE", filepath.Join(topDir, "Antigravity IDE", "Antigravity IDE.exe"), true)
				tryAddDriveExe("Antigravity", filepath.Join(topDir, "Antigravity", "Antigravity.exe"), false)
			}
		}
	}

	return targets
}

func readNullTermANSI(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func readNullTermUTF16LE(b []byte) string {
	var u16 []uint16
	for i := 0; i+1 < len(b); i += 2 {
		v := binary.LittleEndian.Uint16(b[i : i+2])
		if v == 0 {
			break
		}
		u16 = append(u16, v)
	}
	return string(utf16.Decode(u16))
}

func expandWindowsEnvVars(s string) string {
	return winEnvVarPattern.ReplaceAllStringFunc(s, func(m string) string {
		key := m[1 : len(m)-1]
		if val := os.Getenv(key); val != "" {
			return val
		}
		return m
	})
}

// parseWindowsLnkTarget parses a Windows Shell Link (.lnk) file in pure Go ([MS-SHLLINK])
// without spawning PowerShell or WScript.Shell COM objects.
func parseWindowsLnkTarget(lnkPath string) string {
	b, err := os.ReadFile(lnkPath)
	if err != nil || len(b) < 76 || len(b) > 2*1024*1024 {
		return ""
	}
	if binary.LittleEndian.Uint32(b[0:4]) != 0x0000004C {
		return ""
	}
	// Verify ShellLink CLSID prefix 00021401-0000-0000-C000-000000000046
	if binary.LittleEndian.Uint32(b[4:8]) != 0x00021401 {
		return ""
	}

	linkFlags := binary.LittleEndian.Uint32(b[20:24])
	isUnicode := (linkFlags & 0x00000080) != 0
	pos := 76

	// 1. LinkTargetIDList
	if (linkFlags & 0x00000001) != 0 {
		if pos+2 > len(b) {
			return ""
		}
		idListSize := int(binary.LittleEndian.Uint16(b[pos : pos+2]))
		pos += 2 + idListSize
		if pos > len(b) {
			return ""
		}
	}

	var localTarget string

	// 2. LinkInfo
	if (linkFlags & 0x00000002) != 0 {
		if pos+4 > len(b) {
			return ""
		}
		linkInfoSize := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
		if linkInfoSize < 28 || pos+linkInfoSize > len(b) {
			return ""
		}
		infoBlock := b[pos : pos+linkInfoSize]
		headerSize := int(binary.LittleEndian.Uint32(infoBlock[4:8]))
		infoFlags := binary.LittleEndian.Uint32(infoBlock[8:12])

		if (infoFlags & 0x00000001) != 0 {
			var basePath, commonSuffix string
			if headerSize >= 0x24 && len(infoBlock) >= 36 {
				uBaseOff := int(binary.LittleEndian.Uint32(infoBlock[28:32]))
				uSufOff := int(binary.LittleEndian.Uint32(infoBlock[32:36]))
				if uBaseOff > 0 && uBaseOff < len(infoBlock) {
					basePath = readNullTermUTF16LE(infoBlock[uBaseOff:])
				}
				if uSufOff > 0 && uSufOff < len(infoBlock) {
					commonSuffix = readNullTermUTF16LE(infoBlock[uSufOff:])
				}
			}
			if basePath == "" {
				aBaseOff := int(binary.LittleEndian.Uint32(infoBlock[16:20]))
				aSufOff := int(binary.LittleEndian.Uint32(infoBlock[24:28]))
				if aBaseOff > 0 && aBaseOff < len(infoBlock) {
					basePath = readNullTermANSI(infoBlock[aBaseOff:])
				}
				if aSufOff > 0 && aSufOff < len(infoBlock) {
					commonSuffix = readNullTermANSI(infoBlock[aSufOff:])
				}
			}
			if basePath != "" {
				if commonSuffix != "" {
					localTarget = filepath.Join(basePath, commonSuffix)
				} else {
					localTarget = basePath
				}
			}
		}
		pos += linkInfoSize
	}

	readStringData := func() (string, bool) {
		if pos+2 > len(b) {
			return "", false
		}
		charCount := int(binary.LittleEndian.Uint16(b[pos : pos+2]))
		pos += 2
		if isUnicode {
			byteCount := charCount * 2
			if pos+byteCount > len(b) {
				return "", false
			}
			u16 := make([]uint16, charCount)
			for i := 0; i < charCount; i++ {
				u16[i] = binary.LittleEndian.Uint16(b[pos+i*2 : pos+i*2+2])
			}
			pos += byteCount
			return string(utf16.Decode(u16)), true
		}
		if pos+charCount > len(b) {
			return "", false
		}
		s := string(b[pos : pos+charCount])
		pos += charCount
		return s, true
	}

	var relPath string
	// HasName (0x04), HasRelativePath (0x08), HasWorkingDir (0x10), HasArguments (0x20), HasIconLocation (0x40)
	for _, flagBit := range []uint32{0x04, 0x08, 0x10, 0x20, 0x40} {
		if (linkFlags & flagBit) != 0 {
			s, ok := readStringData()
			if !ok {
				break
			}
			if flagBit == 0x08 {
				relPath = s
			}
		}
	}

	// ExtraData blocks (check EnvironmentVariableDataBlock 0xA0000001)
	var envTarget string
	for pos+8 <= len(b) {
		blkSize := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
		if blkSize < 4 || pos+blkSize > len(b) {
			break
		}
		sig := binary.LittleEndian.Uint32(b[pos+4 : pos+8])
		if sig == 0xA0000001 && blkSize >= 788 {
			uTarget := readNullTermUTF16LE(b[pos+268 : pos+788])
			if uTarget == "" {
				uTarget = readNullTermANSI(b[pos+8 : pos+268])
			}
			if uTarget != "" {
				envTarget = expandWindowsEnvVars(uTarget)
			}
		}
		pos += blkSize
	}

	for _, cand := range []string{localTarget, envTarget} {
		if cand != "" {
			cleaned := filepath.Clean(cand)
			if _, err := os.Stat(cleaned); err == nil {
				return cleaned
			}
		}
	}
	if relPath != "" {
		relCand := filepath.Clean(filepath.Join(filepath.Dir(lnkPath), relPath))
		if _, err := os.Stat(relCand); err == nil {
			return relCand
		}
	}
	return ""
}

// ResolveWindowsShortcut resolves a .lnk file to its actual target path in pure Go.
func ResolveWindowsShortcut(lnkPath string) string {
	if !strings.HasSuffix(strings.ToLower(lnkPath), ".lnk") {
		return lnkPath
	}
	if resolved := parseWindowsLnkTarget(lnkPath); resolved != "" {
		return resolved
	}
	return lnkPath
}

type comVtbl struct {
	methods [32]uintptr
}

type comObject struct {
	vtbl *comVtbl
}

// CreateNativeWindowsShortcut creates a standard Windows .lnk shortcut directly via OLE32 IShellLinkW
// without spawning PowerShell, WScript.Shell, or intermediate .bat/.ps1 wrappers.
func CreateNativeWindowsShortcut(lnkPath, targetPath, args, workDir, iconLocation string, iconIndex int, description string, showCmd int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const (
		coinitApartmentThreaded = 0x2
		clsctxInprocServer      = 0x1
	)

	hr, _, _ := procCoInitializeLink.Call(0, coinitApartmentThreaded)
	if hr == 0 || hr == 1 {
		defer procCoUninitLink.Call()
	}

	var pShellLink *comObject
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIShellLinkW)),
		uintptr(unsafe.Pointer(&pShellLink)),
	)
	if hr != 0 || pShellLink == nil {
		return fmt.Errorf("CoCreateInstance(IShellLinkW) failed: 0x%x", hr)
	}
	defer syscall.SyscallN(pShellLink.vtbl.methods[2], uintptr(unsafe.Pointer(pShellLink))) // Release

	setStr := func(methodIdx int, val string) error {
		ptr, err := syscall.UTF16PtrFromString(val)
		if err != nil {
			return err
		}
		r, _, _ := syscall.SyscallN(pShellLink.vtbl.methods[methodIdx], uintptr(unsafe.Pointer(pShellLink)), uintptr(unsafe.Pointer(ptr)))
		if r != 0 {
			return fmt.Errorf("IShellLinkW method %d failed: 0x%x", methodIdx, r)
		}
		return nil
	}

	// IShellLinkW vtable: 20=SetPath, 11=SetArguments, 9=SetWorkingDirectory, 7=SetDescription, 15=SetShowCmd, 17=SetIconLocation
	if err := setStr(20, targetPath); err != nil {
		return err
	}
	if args != "" {
		_ = setStr(11, args)
	}
	if workDir != "" {
		_ = setStr(9, workDir)
	}
	if description != "" {
		_ = setStr(7, description)
	}
	if showCmd > 0 {
		_, _, _ = syscall.SyscallN(pShellLink.vtbl.methods[15], uintptr(unsafe.Pointer(pShellLink)), uintptr(showCmd))
	}
	if iconLocation != "" {
		if iconPtr, err := syscall.UTF16PtrFromString(iconLocation); err == nil {
			_, _, _ = syscall.SyscallN(
				pShellLink.vtbl.methods[17],
				uintptr(unsafe.Pointer(pShellLink)),
				uintptr(unsafe.Pointer(iconPtr)),
				uintptr(iconIndex),
			)
		}
	}

	// QueryInterface(IID_IPersistFile) -> method 0
	var pPersistFile *comObject
	hr, _, _ = syscall.SyscallN(
		pShellLink.vtbl.methods[0],
		uintptr(unsafe.Pointer(pShellLink)),
		uintptr(unsafe.Pointer(&iidIPersistFile)),
		uintptr(unsafe.Pointer(&pPersistFile)),
	)
	if hr != 0 || pPersistFile == nil {
		return fmt.Errorf("QueryInterface(IPersistFile) failed: 0x%x", hr)
	}
	defer syscall.SyscallN(pPersistFile.vtbl.methods[2], uintptr(unsafe.Pointer(pPersistFile))) // Release

	lnkPtr, err := syscall.UTF16PtrFromString(lnkPath)
	if err != nil {
		return err
	}
	// IPersistFile::Save -> method 6
	hr, _, _ = syscall.SyscallN(
		pPersistFile.vtbl.methods[6],
		uintptr(unsafe.Pointer(pPersistFile)),
		uintptr(unsafe.Pointer(lnkPtr)),
		1,
	)
	if hr != 0 {
		return fmt.Errorf("IPersistFile.Save failed: 0x%x", hr)
	}
	return nil
}

func quoteWindowsArg(a string) string {
	if a == "" {
		return `""`
	}
	if !strings.ContainsAny(a, " \t\"") {
		return a
	}
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}

func launchElevatedPlatform(appPath string, args []string) error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	exePtr, err := syscall.UTF16PtrFromString(appPath)
	if err != nil {
		return err
	}
	var quoted []string
	for _, a := range args {
		quoted = append(quoted, quoteWindowsArg(a))
	}
	var argPtr *uint16
	if len(quoted) > 0 {
		argPtr, _ = syscall.UTF16PtrFromString(strings.Join(quoted, " "))
	}
	dirPtr, _ := syscall.UTF16PtrFromString(filepath.Dir(appPath))
	return windows.ShellExecute(0, verb, exePtr, argPtr, dirPtr, windows.SW_SHOWNORMAL)
}
