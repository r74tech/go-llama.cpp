#!/bin/bash
set -e

echo "=== Testing modelembed with temporary file monitoring ==="

# Create models directory
mkdir -p models

# Check if model already exists
if [ ! -f models/model.gguf ]; then
    echo "Downloading test model..."
    wget -q --show-progress https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf -O models/model.gguf
fi

# Build the binary
echo "Building modelembed..."
go build -o modelembed main.go

# Function to monitor file system for temporary files
monitor_tempfiles() {
    local pid=$1
    # Set temp directories based on OS
    local temp_dirs
    if [ "$(uname)" = "Linux" ] || [ "$(uname)" = "Darwin" ]; then
        temp_dirs=("/tmp" "/var/tmp" "$TMPDIR")
    else
        # Windows
        temp_dirs=("$TEMP" "$TMP" "C:\\Windows\\Temp" "C:\\Temp")
    fi
    local found_tempfile=0
    
    echo "Monitoring for temporary files (PID: $pid)..."
    
    while kill -0 $pid 2>/dev/null; do
        for dir in "${temp_dirs[@]}"; do
            if [ -d "$dir" ]; then
                # Look for files created by our process
                if [ "$(uname)" = "Linux" ]; then
                    # On Linux, check /proc/PID/fd for open files
                    if [ -d "/proc/$pid/fd" ]; then
                        for fd in /proc/$pid/fd/*; do
                            if [ -L "$fd" ]; then
                                link=$(readlink "$fd")
                                if [[ "$link" == *"/tmp/"* ]] || [[ "$link" == *"llama"* ]]; then
                                    if [[ "$link" != *"/memfd:"* ]] && [[ "$link" != *"(deleted)"* ]]; then
                                        echo "WARNING: Found temporary file: $link"
                                        found_tempfile=1
                                    fi
                                fi
                            fi
                        done
                    fi
                fi
                
                # Also check for llama-related files in temp directories
                find "$dir" -name "*llama*" -type f -newer models/model.gguf 2>/dev/null | while read -r file; do
                    echo "WARNING: Found temporary file: $file"
                    found_tempfile=1
                done
            fi
        done
        sleep 0.1
    done
    
    return $found_tempfile
}

# Test embedded model with monitoring
echo -e "\n=== Testing embedded model (monitoring for temp files) ==="

# Start the model in background
./modelembed -embedded -i=false -n 50 -t 4 "Once upon a time" &
MODEL_PID=$!

# Monitor for temporary files
monitor_tempfiles $MODEL_PID &
MONITOR_PID=$!

# Wait for model to complete
wait $MODEL_PID
MODEL_EXIT_CODE=$?

# Stop monitoring
kill $MONITOR_PID 2>/dev/null || true
wait $MONITOR_PID 2>/dev/null || true

if [ $MODEL_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Model execution failed"
    exit 1
fi

# Additional checks for Linux
if [ "$(uname)" = "Linux" ]; then
    echo -e "\n=== Linux-specific checks ==="
    
    # Check if memfd_create was used by looking at strace
    if command -v strace >/dev/null 2>&1; then
        echo "Running with strace to verify memfd_create usage..."
        strace -e trace=memfd_create,open,openat,creat -o strace.log ./modelembed -embedded -i=false -n 20 -t 4 "Test"
        
        if grep -q "memfd_create" strace.log; then
            echo "✓ memfd_create was called (good - using in-memory file)"
        else
            echo "✗ memfd_create was NOT called (unexpected)"
        fi
        
        echo -e "\nChecking for file operations in /tmp:"
        if grep -E "open.*(/tmp/|llama)" strace.log | grep -v "ENOENT"; then
            echo "✗ Found file operations in /tmp (unexpected)"
        else
            echo "✓ No file operations in /tmp (good)"
        fi
        
        rm -f strace.log
    fi
fi

# Test on Windows
if [ "$(uname -o 2>/dev/null)" = "Msys" ] || [ "$(uname -o 2>/dev/null)" = "Cygwin" ] || [ "$OS" = "Windows_NT" ]; then
    echo -e "\n=== Windows-specific checks ==="
    echo "Note: On Windows, FILE_ATTRIBUTE_TEMPORARY keeps files in cache"
    echo "Checking Windows temp directories..."
    
    # Build list of Windows temp directories
    WIN_TEMP_DIRS=""
    [ -d "$TEMP" ] && WIN_TEMP_DIRS="$TEMP"
    [ -d "$TMP" ] && [ "$TMP" != "$TEMP" ] && WIN_TEMP_DIRS="$WIN_TEMP_DIRS $TMP"
    [ -d "C:/Windows/Temp" ] && WIN_TEMP_DIRS="$WIN_TEMP_DIRS C:/Windows/Temp"
    
    echo "Temp directories: $WIN_TEMP_DIRS"
    TEMP_COUNT_BEFORE=$(find $WIN_TEMP_DIRS -name "*llama*" -type f 2>/dev/null | wc -l)
    
    ./modelembed -embedded -i=false -n 20 -t 4 "Test"
    
    TEMP_COUNT_AFTER=$(find $WIN_TEMP_DIRS -name "*llama*" -type f 2>/dev/null | wc -l)
    
    if [ $TEMP_COUNT_AFTER -gt $TEMP_COUNT_BEFORE ]; then
        echo "✗ New temporary files were created"
    else
        echo "✓ No new temporary files detected"
    fi
fi

echo -e "\n=== Memory usage check ==="
# Get the model file size
MODEL_SIZE=$(stat -c%s models/model.gguf 2>/dev/null || stat -f%z models/model.gguf 2>/dev/null)
MODEL_SIZE_MB=$((MODEL_SIZE / 1024 / 1024))
echo "Model size: ${MODEL_SIZE_MB} MB"

# Check binary size to confirm embedding
BINARY_SIZE=$(stat -c%s modelembed 2>/dev/null || stat -f%z modelembed 2>/dev/null)
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))
echo "Binary size: ${BINARY_SIZE_MB} MB"

if [ $BINARY_SIZE_MB -ge $MODEL_SIZE_MB ]; then
    echo "✓ Binary size suggests model is embedded"
else
    echo "✗ Binary size too small - model may not be embedded"
fi

echo -e "\nAll tests completed!"