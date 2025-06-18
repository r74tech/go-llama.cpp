package main

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/r74tech/go-llama.cpp"
)

// Embed a model file into the binary
//
//go:embed model.gguf
var modelData []byte

func main() {
	// Load model from embedded data
	fmt.Println("Loading model from memory...")
	model, err := llama.NewFromMemory(modelData, llama.SetContext(128))
	if err != nil {
		log.Fatalf("Failed to load model from memory: %v", err)
	}
	defer model.Free()

	// Test the model
	fmt.Println("Model loaded successfully from memory!")

	// Try to predict something
	response, err := model.Predict("Hello, ", llama.SetTemperature(0.8))
	if err != nil {
		log.Fatalf("Failed to predict: %v", err)
	}

	fmt.Printf("Response: %s\n", response)
}
