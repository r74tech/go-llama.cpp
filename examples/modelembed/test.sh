#!/bin/bash
set -e

echo "=== Testing modelembed example ==="

# Create models directory
mkdir -p models

# Check if model already exists
if [ ! -f models/model.gguf ]; then
    echo "Downloading test model..."
    wget -q --show-progress https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf -O models/model.gguf
else
    echo "Model already exists, skipping download"
fi

# Check model file
echo "Model info:"
ls -lh models/model.gguf
file models/model.gguf

# Build the binary
echo "Building modelembed..."
go build -o modelembed main.go

# Test embedded model
echo -e "\n=== Testing embedded model ==="
./modelembed -embedded -i=false -n 50 -t 4 "Once upon a time"

# Test file-based model
echo -e "\n=== Testing file-based model ==="
./modelembed -m models/model.gguf -i=false -n 50 -t 4 "Once upon a time"

echo -e "\nAll tests completed successfully!"