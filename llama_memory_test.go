package llama

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestMemoryLeakInPredict tests for memory leaks in the Predict function
func TestMemoryLeakInPredict(t *testing.T) {
	// Skip if no model is available
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	
	model, err := New("ggllm-test-model.bin", EnableF16Memory, SetContext(128))
	if err != nil {
		t.Skip("Test model not found, skipping memory leak test")
	}
	defer model.Free()
	
	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	// Run multiple predictions
	for i := 0; i < 10; i++ {
		_, err := model.Predict("Hello world", SetTokens(10))
		if err != nil {
			t.Fatalf("Prediction failed: %v", err)
		}
		
		// Force GC between iterations
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	
	// Get final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	// Check for significant memory growth
	memGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	if memGrowth > 10*1024*1024 { // 10MB threshold
		t.Errorf("Possible memory leak detected: memory grew by %d bytes", memGrowth)
	}
}

// TestLargeInputBuffer tests handling of large input strings
func TestLargeInputBuffer(t *testing.T) {
	model, err := New("ggllm-test-model.bin", EnableF16Memory, SetContext(512))
	if err != nil {
		t.Skip("Test model not found, skipping large input test")
	}
	defer model.Free()
	
	// Create a very large input string
	largeInput := strings.Repeat("Hello world ", 10000)
	
	// This should not crash or cause memory issues
	_, err = model.Predict(largeInput, SetTokens(10))
	if err != nil {
		// This is expected - the input is too large
		t.Logf("Expected error for large input: %v", err)
	}
}

// TestConcurrentPredictions tests thread safety and memory management
func TestConcurrentPredictions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}
	
	model, err := New("ggllm-test-model.bin", EnableF16Memory, SetContext(128))
	if err != nil {
		t.Skip("Test model not found, skipping concurrent test")
	}
	defer model.Free()
	
	// Run multiple goroutines making predictions
	// With the mutex, these should serialize properly
	done := make(chan bool, 3)
	errChan := make(chan error, 3)
	
	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("Goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()
			
			_, err := model.Predict("Hello", SetTokens(5))
			if err != nil {
				errChan <- fmt.Errorf("Goroutine %d prediction failed: %v", id, err)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
	
	close(errChan)
	for err := range errChan {
		t.Error(err)
	}
}

// TestEmptyInput tests handling of empty input strings
func TestEmptyInput(t *testing.T) {
	model, err := New("ggllm-test-model.bin", EnableF16Memory, SetContext(128))
	if err != nil {
		t.Skip("Test model not found, skipping empty input test")
	}
	defer model.Free()
	
	// Test with empty string
	result, err := model.Predict("", SetTokens(10))
	if err != nil {
		t.Fatalf("Failed to handle empty input: %v", err)
	}
	
	// The model should generate some output even with empty input
	if len(result) == 0 {
		t.Error("Expected some output for empty input")
	}
}

// TestBufferBoundary tests edge cases around buffer boundaries
func TestBufferBoundary(t *testing.T) {
	model, err := New("ggllm-test-model.bin", EnableF16Memory, SetContext(128))
	if err != nil {
		t.Skip("Test model not found, skipping buffer boundary test")
	}
	defer model.Free()
	
	// Test with strings of various sizes near typical buffer boundaries
	sizes := []int{255, 256, 257, 1023, 1024, 1025, 4095, 4096, 4097}
	
	for _, size := range sizes {
		input := strings.Repeat("a", size)
		_, err := model.Predict(input, SetTokens(5))
		if err != nil {
			t.Logf("Size %d: %v", size, err)
		}
	}
}

// TestMemoryFromBuffer tests loading model from memory buffer
func TestMemoryFromBuffer(t *testing.T) {
	// Read model file into memory
	// This is a placeholder - implement when you have a test model
	t.Skip("TestMemoryFromBuffer not implemented yet")
}