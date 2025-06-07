# Using Pre-built Libraries

This project supports using pre-built libraries to avoid compiling llama.cpp from source each time.

## What's in the release?

Each release package contains:
- `libbinding.a` - Pre-built static library (with patches already applied)
- All necessary Go source files
- C/C++ binding files
- `setup.sh` - Script to set up llama.cpp submodule at the correct commit
- `LLAMA_COMMIT` - File containing the exact llama.cpp commit used

## How to use pre-built libraries

### Method 1: Complete package download (Simplest)

```bash
# 1. Download and extract the release for your platform
curl -L -o libbinding.tar.gz https://github.com/r74tech/go-llama.cpp/releases/download/v0.1.0/libbinding-linux-amd64.tar.gz
tar -xzf libbinding.tar.gz

# 2. Run the setup script to get llama.cpp headers
./setup.sh

# 3. Build your application
go build
```

### Method 2: Using existing repository with pre-built library

```bash
# Clone repository
git clone https://github.com/r74tech/go-llama.cpp
cd go-llama.cpp

# Download pre-built library using build tag
go build -tags prebuilt

# Or specify a specific release
LLAMA_CPP_RELEASE_TAG=v0.1.0 go build -tags prebuilt
```

### Method 3: Manual setup

```bash
# If you already have the repository
cd go-llama.cpp

# Run the download script
go run scripts/download-libs.go

# Make sure llama.cpp is at the correct commit
cd llama.cpp
git fetch --depth 1 origin <commit-from-release>
git checkout <commit-from-release>
cd ..

# Build
go build
```

## Important Notes

- The patches are already applied to `libbinding.a`, you don't need to apply them manually
- The llama.cpp submodule must be at the exact commit specified in the release
- The `setup.sh` script automatically handles the shallow clone of llama.cpp

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