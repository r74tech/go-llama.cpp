# Memory Loading Example

This example demonstrates how to load a LLaMA model from memory instead of a file path.

## Prerequisites

- A GGUF model file (e.g., `model.gguf`)
- Go 1.16 or later (for embed directive support)

## Usage

1. Place your GGUF model file in this directory and name it `model.gguf`
2. Run the example:

```bash
go run main.go
```

## How it works

The example uses Go's `//go:embed` directive to embed the model file directly into the binary. This allows you to create a single executable that contains both your code and the model.

```go
//go:embed model.gguf
var modelData []byte
```

Then, instead of using `llama.New()` with a file path, we use `llama.NewFromMemory()`:

```go
model, err := llama.NewFromMemory(modelData, llama.SetContext(128))
```

## Note

Currently, this feature requires extending the llama.cpp library to support memory-based model loading. The implementation in this example shows the API design, but the actual memory loading functionality needs to be implemented in the underlying C++ library.
