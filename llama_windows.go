//go:build windows
// +build windows

package llama

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateFileMapping = modkernel32.NewProc("CreateFileMappingW")
	procMapViewOfFile     = modkernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile   = modkernel32.NewProc("UnmapViewOfFile")
)

const (
	PAGE_READONLY = 0x02
	FILE_MAP_READ = 0x04
)

// mmapModel maps a file region into memory (Windows)
func mmapModel(fd int, offset int64, size int) (uintptr, []byte, error) {
	// Get Windows handle from fd
	handle := syscall.Handle(fd)

	// Create file mapping
	// For large files, we need to specify the high and low parts of the size
	maxSizeHigh := uint32((offset + int64(size)) >> 32)
	maxSizeLow := uint32((offset + int64(size)) & 0xFFFFFFFF)

	mapping, _, err := procCreateFileMapping.Call(
		uintptr(handle),
		0, // no security attributes
		PAGE_READONLY,
		uintptr(maxSizeHigh),
		uintptr(maxSizeLow),
		0, // no name
	)

	if mapping == 0 {
		return 0, nil, fmt.Errorf("CreateFileMapping failed: %v", err)
	}

	// Map view of file
	offsetHigh := uint32(offset >> 32)
	offsetLow := uint32(offset & 0xFFFFFFFF)

	addr, _, err := procMapViewOfFile.Call(
		mapping,
		FILE_MAP_READ,
		uintptr(offsetHigh),
		uintptr(offsetLow),
		uintptr(size),
	)

	if addr == 0 {
		syscall.CloseHandle(syscall.Handle(mapping))
		return 0, nil, fmt.Errorf("MapViewOfFile failed: %v", err)
	}

	// Create a slice from the mapped memory using unsafe.Slice (Go 1.17+)
	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), size)

	return addr, data, nil
}

// unmapModel unmaps the memory (Windows specific)
func unmapModel(addr uintptr) error {
	ret, _, err := procUnmapViewOfFile.Call(addr)
	if ret == 0 {
		return fmt.Errorf("UnmapViewOfFile failed: %v", err)
	}
	return nil
}
