# [![Go Reference](https://pkg.go.dev/badge/github.com/go-skynet/go-llama.cpp.svg)](https://pkg.go.dev/github.com/go-skynet/go-llama.cpp) go-llama.cpp (Fork with Memory Buffer Support)

[LLama.cpp](https://github.com/ggerganov/llama.cpp) golang bindings.

**This is a fork of [go-skynet/go-llama.cpp](https://github.com/go-skynet/go-llama.cpp) that adds support for loading models from memory buffers.**

The go-llama.cpp bindings are high level, as such most of the work is kept into the C/C++ code to avoid any extra computational cost, be more performant and lastly ease out maintenance, while keeping the usage as simple as possible.

Check out [this](https://about.sourcegraph.com/blog/go/gophercon-2018-adventures-in-cgo-performance) and [this](https://www.cockroachlabs.com/blog/the-cost-and-complexity-of-cgo/) write-ups which summarize the impact of a low-level interface which calls C functions from Go.

If you are looking for an high-level OpenAI compatible API, check out [here](https://github.com/go-skynet/llama-cli).

## Attention!

Since https://github.com/go-skynet/go-llama.cpp/pull/180 is merged, now go-llama.cpp is not anymore compatible with `ggml` format, but it works ONLY with the new `gguf` file format. See also the upstream PR: https://github.com/ggerganov/llama.cpp/pull/2398.

If you need to use the `ggml` format, use the https://github.com/go-skynet/go-llama.cpp/releases/tag/pre-gguf tag.

## Fork Features

### Memory Buffer Loading

This fork adds the ability to load GGUF models directly from memory buffers, eliminating the need for temporary files. This is useful for:

- Loading models from embedded resources
- Working with models stored in databases or object storage
- Reducing disk I/O operations
- Improving security by keeping models in memory

### Example Usage

<details>
<summary><b>Loading from a byte slice</b></summary>

```go
// Load model from byte slice
modelData, err := ioutil.ReadFile("model.gguf")
if err != nil {
    panic(err)
}

model, err := llama.NewFromMemory(modelData, 
    llama.EnableF16Memory,
    llama.SetContext(128),
    llama.EnableEmbeddings,
    llama.SetGPULayers(0))
if err != nil {
    panic(err)
}
defer model.Free()
```
</details>

<details>
<summary><b>Loading from embedded resources</b></summary>

```go
package main

import (
    _ "embed"
    llama "github.com/go-skynet/go-llama.cpp"
)

// Embed the model file at compile time
//go:embed models/model.gguf
var embeddedModel []byte

func main() {
    // Load model from embedded bytes
    model, err := llama.NewFromMemory(embeddedModel,
        llama.EnableF16Memory,
        llama.SetContext(128),
        llama.EnableEmbeddings,
        llama.SetGPULayers(0))
    if err != nil {
        panic(err)
    }
    defer model.Free()
    
    // Use the model for inference
    _, err = model.Predict("Hello, world!", 
        llama.SetTokens(128),
        llama.SetThreads(4))
}
```
</details>

See the [examples/modelembed](examples/modelembed/main.go) directory for a complete working example with interactive mode.

## Usage

Note: This repository uses git submodules to keep track of [LLama.cpp](https://github.com/ggerganov/llama.cpp).

Clone the repository locally:

```bash
git clone --recurse-submodules https://github.com/go-skynet/go-llama.cpp
```

To build the bindings locally, run:

```
cd go-llama.cpp
make libbinding.a
```

Now you can run the example with:

```
LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go run ./examples -m "/model/path/here" -t 14
```

## Acceleration

### OpenBLAS

To build and run with OpenBLAS, for example:

```
BUILD_TYPE=openblas make libbinding.a
CGO_LDFLAGS="-lopenblas" LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go run -tags openblas ./examples -m "/model/path/here" -t 14
```

### CuBLAS

To build with CuBLAS:

```
BUILD_TYPE=cublas make libbinding.a
CGO_LDFLAGS="-lcublas -lcudart -L/usr/local/cuda/lib64/" LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go run ./examples -m "/model/path/here" -t 14
```

### ROCM

To build with ROCM (HIPBLAS):

```
BUILD_TYPE=hipblas make libbinding.a
CC=/opt/rocm/llvm/bin/clang CXX=/opt/rocm/llvm/bin/clang++ CGO_LDFLAGS="-O3 --hip-link --rtlib=compiler-rt -unwindlib=libgcc -lrocblas -lhipblas" LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go run ./examples -m "/model/path/here" -ngl 64 -t 32
```

### OpenCL

```
BUILD_TYPE=clblas CLBLAS_DIR=... make libbinding.a
CGO_LDFLAGS="-lOpenCL -lclblast -L/usr/local/lib64/" LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go run ./examples -m "/model/path/here" -t 14
```


You should see something like this from the output when using the GPU:

```
ggml_opencl: selecting platform: 'Intel(R) OpenCL HD Graphics'
ggml_opencl: selecting device: 'Intel(R) Graphics [0x46a6]'
ggml_opencl: device FP16 support: true
```

## GPU offloading

### Metal (Apple Silicon)

```
BUILD_TYPE=metal make libbinding.a
CGO_LDFLAGS="-framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders" LIBRARY_PATH=$PWD C_INCLUDE_PATH=$PWD go build ./examples/main.go
cp build/bin/ggml-metal.metal .
./main -m "/model/path/here" -t 1 -ngl 1
```

Enjoy!

The documentation is available [here](https://pkg.go.dev/github.com/go-skynet/go-llama.cpp) and the full example code is [here](https://github.com/go-skynet/go-llama.cpp/blob/master/examples/main.go).

## License

MIT
