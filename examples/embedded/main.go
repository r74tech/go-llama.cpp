package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"

	llama "github.com/go-skynet/go-llama.cpp"
)

// Embed the model file at compile time
// Replace this with your actual model file path
//go:embed model.gguf
var embeddedModel []byte

func main() {
	var (
		threads    = flag.Int("t", runtime.NumCPU(), "number of threads to use during computation")
		tokens     = flag.Int("n", 128, "number of tokens to predict")
		prompt     = flag.String("p", "Hello, world!", "prompt to generate text from")
		contextSize = flag.Int("c", 512, "context size")
		temperature = flag.Float64("temp", 0.8, "temperature for sampling")
		topK       = flag.Int("top_k", 40, "top-k sampling")
		topP       = flag.Float64("top_p", 0.95, "top-p sampling")
		gpuLayers  = flag.Int("ngl", 0, "number of layers to store in VRAM")
	)
	flag.Parse()

	// Check if we're on a supported platform
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		log.Fatal("This embedded model example only supports Linux and Windows")
	}

	// Load model from embedded bytes
	fmt.Printf("Loading embedded model (size: %.2f MB)...\n", float64(len(embeddedModel))/(1024*1024))
	
	model, err := llama.NewFromMemory(embeddedModel, llama.SetContext(*contextSize), llama.SetGPULayers(*gpuLayers))
	if err != nil {
		log.Fatalf("Failed to load model from memory: %v", err)
	}
	defer model.Free()

	fmt.Println("Model loaded successfully from embedded data!")

	// Generate text
	fmt.Printf("\nPrompt: %s\n", *prompt)
	fmt.Println("\nGenerating response...\n")

	var response strings.Builder
	
	_, err = model.Predict(*prompt, llama.SetTokens(*tokens), llama.SetThreads(*threads),
		llama.SetTemperature(float32(*temperature)),
		llama.SetTopK(*topK),
		llama.SetTopP(float32(*topP)),
		llama.SetTokenCallback(func(token string) bool {
			response.WriteString(token)
			fmt.Print(token)
			return true
		}))
	
	if err != nil {
		log.Fatalf("Failed to generate text: %v", err)
	}

	fmt.Printf("\n\nComplete response:\n%s\n", response.String())
}