# Embedded Model Example

This example demonstrates how to embed a GGUF model directly into a Go binary using `go:embed` and run it from memory without creating temporary files.

## Requirements

- Linux or Windows (macOS is not supported for memory loading)
- A GGUF format model file
- Go 1.16 or later (for embed support)

## Setup

1. Copy your GGUF model file to this directory and rename it to `model.gguf`:
   ```bash
   cp /path/to/your/model.gguf model.gguf
   ```

2. Build the binary with the embedded model:
   ```bash
   go build -o embedded-llama main.go
   ```

3. Run the executable:
   ```bash
   ./embedded-llama -p "Once upon a time" -n 100
   ```

## Command Line Options

- `-p`: Prompt to generate text from (default: "Hello, world!")
- `-n`: Number of tokens to predict (default: 128)
- `-t`: Number of threads to use (default: number of CPU cores)
- `-c`: Context size (default: 512)
- `-temp`: Temperature for sampling (default: 0.8)
- `-top_k`: Top-k sampling (default: 40)
- `-top_p`: Top-p sampling (default: 0.95)
- `-ngl`: Number of layers to store in VRAM for GPU acceleration (default: 0)

## How It Works

1. The model file is embedded into the binary at compile time using Go's `//go:embed` directive
2. On Linux, the model is loaded using `memfd_create()` to create an in-memory file descriptor
3. On Windows, a memory-mapped file is used to avoid disk I/O
4. The model runs entirely from memory without creating temporary files on disk

## Notes

- The resulting binary will be large (model size + program size)
- Initial load time may be longer due to the embedded data extraction
- This approach is ideal for distributing self-contained AI applications
- Make sure you have enough RAM to hold the model in memory