# Using Pre-built Libraries

This project supports using pre-built libraries to avoid compiling llama.cpp from source each time.

## How to use pre-built libraries

### Method 1: Using build tags (Recommended)

```bash
# Download pre-built libraries and build
go build -tags prebuilt

# Or specify a specific release
LLAMA_CPP_RELEASE_TAG=v0.1.0 go build -tags prebuilt
```

### Method 2: Manual download

```bash
# Run the download script directly
go run scripts/download-libs.go

# Then build normally
go build
```

## Building and releasing pre-built libraries

The GitHub Actions workflow automatically builds libraries for multiple platforms when you create a new tag:

```bash
# Create and push a new tag
git tag v0.1.0
git push origin v0.1.0
```

The workflow will build for:
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

## What's included in pre-built packages

Each pre-built package includes:
- `libbinding.a` - The compiled static library
- `binding.h` - Header file
- `llama.cpp/` directory with required headers:
  - `ggml.h`
  - `ggml-alloc.h`
  - `ggml-backend.h`
  - `ggml-metal.h` (macOS only)
  - `ggml-cuda.h` (if CUDA support is built)
  - `llama.h`

## Customizing the download

You can customize the download behavior by:

1. Setting the repository owner in `scripts/download-libs.go`
2. Using environment variable `LLAMA_CPP_RELEASE_TAG` to specify a release version
3. Modifying the GitHub Actions workflow to include additional build configurations