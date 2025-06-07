// +build prebuilt

package llama

// This build tag enables using pre-built libraries instead of building from source.
// To use pre-built libraries, build with: go build -tags prebuilt

//go:generate go run scripts/download-libs.go

// When using prebuilt tag, we assume libbinding.a is already present
// either from manual placement or from the download script