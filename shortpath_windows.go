package main

import (
	"syscall"
	"unsafe"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procGetShortPathName = modkernel32.NewProc("GetShortPathNameW")
)

// ToShortPath converts a long Unicode path to a short 8.3 path
// so that tools like 3proxy (ANSI-only) can use it on Windows.
// Returns original path on error.
func ToShortPath(longPath string) string {
	if longPath == "" {
		return longPath
	}
	lpszLongPath, err := syscall.UTF16PtrFromString(longPath)
	if err != nil {
		return longPath
	}
	// First call: get required buffer size
	n, _, _ := procGetShortPathName.Call(
		uintptr(unsafe.Pointer(lpszLongPath)),
		0,
		0,
	)
	if n == 0 {
		return longPath
	}
	buf := make([]uint16, n)
	n2, _, _ := procGetShortPathName.Call(
		uintptr(unsafe.Pointer(lpszLongPath)),
		uintptr(unsafe.Pointer(&buf[0])),
		n,
	)
	if n2 == 0 {
		return longPath
	}
	return syscall.UTF16ToString(buf)
}
