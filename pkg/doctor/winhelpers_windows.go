//go:build windows

package doctor

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modUser32              = windows.NewLazySystemDLL("user32.dll")
	procOpenClipboard      = modUser32.NewProc("OpenClipboard")
	procCloseClipboard     = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard     = modUser32.NewProc("EmptyClipboard")
	procSetClipboardData   = modUser32.NewProc("SetClipboardData")
	modKernel32Clip        = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc        = modKernel32Clip.NewProc("GlobalAlloc")
	procGlobalFree         = modKernel32Clip.NewProc("GlobalFree")
	procGlobalLock         = modKernel32Clip.NewProc("GlobalLock")
	procGlobalUnlock       = modKernel32Clip.NewProc("GlobalUnlock")
)

func detectWindowsOSInfoNative(arch string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return fmt.Sprintf("Windows [%s]", arch)
	}
	defer k.Close()

	prodName, _, _ := k.GetStringValue("ProductName")
	dispVer, _, _ := k.GetStringValue("DisplayVersion")
	buildNum, _, _ := k.GetStringValue("CurrentBuildNumber")
	if buildNum == "" {
		buildNum, _, _ = k.GetStringValue("CurrentBuild")
	}

	// Windows 11 builds >= 22000 still report "Windows 10" in legacy ProductName
	if strings.Contains(prodName, "Windows 10") && buildNum >= "22000" {
		prodName = strings.Replace(prodName, "Windows 10", "Windows 11", 1)
	}

	var parts []string
	if prodName != "" {
		parts = append(parts, prodName)
	} else {
		parts = append(parts, "Windows")
	}
	if dispVer != "" {
		parts = append(parts, dispVer)
	}
	if buildNum != "" {
		parts = append(parts, "(Build "+buildNum+")")
	}
	return fmt.Sprintf("%s [%s]", strings.Join(parts, " "), arch)
}

func copyToWindowsClipboardNative(text string) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	utf16Data, err := syscall.UTF16FromString(text)
	if err != nil || len(utf16Data) == 0 {
		return false
	}

	const (
		gmemMoveable  = 0x0002
		cfUnicodeText = 13
	)

	byteLen := uintptr(len(utf16Data) * 2)
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, byteLen)
	if hMem == 0 {
		return false
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procGlobalFree.Call(hMem)
		return false
	}

	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16Data))
	copy(dst, utf16Data)
	procGlobalUnlock.Call(hMem)

	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		procGlobalFree.Call(hMem)
		return false
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	res, _, _ := procSetClipboardData.Call(cfUnicodeText, hMem)
	if res == 0 {
		procGlobalFree.Call(hMem)
		return false
	}
	return true
}

func openWindowsFileNative(filePath string) bool {
	verb, err1 := syscall.UTF16PtrFromString("open")
	filePtr, err2 := syscall.UTF16PtrFromString(filePath)
	if err1 != nil || err2 != nil {
		return false
	}
	err := windows.ShellExecute(0, verb, filePtr, nil, nil, windows.SW_SHOWNORMAL)
	return err == nil
}
