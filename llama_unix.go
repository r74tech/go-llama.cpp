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
	// Use syscall.Mmap to map the model portion
	data, err := syscall.Mmap(fd, offset, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to mmap model: %w", err)
	}

	// Get the base address of the mmap'd region
	addr := uintptr(unsafe.Pointer(&data[0]))

	return addr, data, nil
}
