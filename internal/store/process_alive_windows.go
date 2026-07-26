//go:build windows

package store

import (
	"syscall"
	"unsafe"
)

const processQueryLimitedInformation = 0x1000
const stillActive = 259

var storeKernel32 = syscall.NewLazyDLL("kernel32.dll")
var storeOpenProcess = storeKernel32.NewProc("OpenProcess")
var storeGetExitCodeProcess = storeKernel32.NewProc("GetExitCodeProcess")
var storeCloseHandle = storeKernel32.NewProc("CloseHandle")

func processAlive(pid int) bool {
	handle, _, callErr := storeOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return callErr == syscall.ERROR_ACCESS_DENIED
	}
	defer storeCloseHandle.Call(handle)
	var code uint32
	result, _, _ := storeGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&code)))
	return result != 0 && code == stillActive
}
