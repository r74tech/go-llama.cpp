//go:build linux || darwin || freebsd || openbsd || netbsd
// +build linux darwin freebsd openbsd netbsd

package llama

import (
	"fmt"
	"syscall"
	"unsafe"
)

// mmapModel maps a file region into memory (Unix systems)
func mmapModel(fd int, offset int64, size int) (uintptr, []byte, error) {
	// mmap requires page-aligned offset
	pageSize := int64(syscall.Getpagesize())
	pageAlignedOffset := (offset / pageSize) * pageSize
	adjustment := int(offset - pageAlignedOffset)
	mapSize := size + adjustment

	// Use syscall.Mmap to map the model portion
	data, err := syscall.Mmap(fd, pageAlignedOffset, mapSize, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to mmap model: %w", err)
	}

	// Return the adjusted address pointing to the actual model data
	// Skip the page alignment padding
	modelData := data[adjustment:]
	addr := uintptr(unsafe.Pointer(&modelData[0]))

	return addr, modelData[:size], nil
}
