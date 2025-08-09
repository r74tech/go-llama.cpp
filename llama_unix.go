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
	
	fmt.Printf("Debug mmap: offset=%d (0x%x), pageAlignedOffset=%d (0x%x), adjustment=%d, mapSize=%d\n",
		offset, offset, pageAlignedOffset, pageAlignedOffset, adjustment, mapSize)

	// Use syscall.Mmap to map the model portion
	data, err := syscall.Mmap(fd, pageAlignedOffset, mapSize, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to mmap model: %w", err)
	}

	// Return the adjusted address pointing to the actual model data
	// Skip the page alignment padding
	modelData := data[adjustment:]
	addr := uintptr(unsafe.Pointer(&modelData[0]))
	
	fmt.Printf("Debug mmap: mapped %d bytes, returning data from offset %d\n", len(data), adjustment)

	return addr, modelData[:size], nil
}
