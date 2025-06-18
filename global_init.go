package llama

import (
	"runtime"
	"runtime/debug"
)

func init() {
	// Set the maximum number of OS threads
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	// Set garbage collection target percentage
	// Lower value means more frequent GC (default is 100)
	debug.SetGCPercent(50)
	
	// Set memory limit if needed (Go 1.19+)
	// This helps prevent OOM by triggering GC more aggressively
	// when approaching the limit
	// debug.SetMemoryLimit(8 * 1024 * 1024 * 1024) // 8GB limit
	
	// Force a GC run at startup to clean up any initial allocations
	runtime.GC()
	runtime.Gosched()
}