package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	llama "github.com/go-skynet/go-llama.cpp"
)

//go:embed models/model.gguf
var embeddedModel []byte

var (
	threads   = 4
	tokens    = 128
	gpulayers = 0
	seed      = -1
)

func main() {
	var model string
	var useEmbedded bool
	var interactive bool

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&model, "m", "./models/7B/ggml-model-q4_0.bin", "path to model file to load")
	flags.BoolVar(&useEmbedded, "embedded", false, "use embedded model instead of file")
	flags.BoolVar(&interactive, "i", true, "interactive mode")
	flags.IntVar(&gpulayers, "ngl", 0, "Number of GPU layers to use")
	flags.IntVar(&threads, "t", runtime.NumCPU(), "number of threads to use during computation")
	flags.IntVar(&tokens, "n", 512, "number of tokens to predict")
	flags.IntVar(&seed, "s", -1, "predict RNG seed, -1 for random seed")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("Parsing program arguments failed: %s\n", err)
		os.Exit(1)
	}

	// モデルのロード
	var l *llama.LLama
	if useEmbedded {
		if len(embeddedModel) == 0 {
			fmt.Println("Error: No embedded model found. Please build with a model file.")
			os.Exit(1)
		}

		fmt.Printf("Loading embedded model from memory (%d MB)...\n", len(embeddedModel)/(1024*1024))
		l, err = llama.NewFromMemory(embeddedModel,
			llama.EnableF16Memory,
			llama.SetContext(128),
			llama.EnableEmbeddings,
			llama.SetGPULayers(gpulayers))
		if err != nil {
			fmt.Println("Loading the model from memory failed:", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Model loaded successfully from memory.\n")
	} else {
		fmt.Printf("Loading model from file: %s\n", model)
		l, err = llama.New(model,
			llama.EnableF16Memory,
			llama.SetContext(128),
			llama.EnableEmbeddings,
			llama.SetGPULayers(gpulayers))
		if err != nil {
			fmt.Println("Loading the model failed:", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Model loaded successfully from file.\n")
	}

	defer l.Free()

	if interactive {
		// インタラクティブモード
		reader := bufio.NewReader(os.Stdin)
		for {
			text := readMultiLineInput(reader)

			_, err := l.Predict(text, llama.Debug, llama.SetTokenCallback(func(token string) bool {
				fmt.Print(token)
				return true
			}), llama.SetTokens(tokens), llama.SetThreads(threads), llama.SetTopK(90), llama.SetTopP(0.86), llama.SetStopWords("llama"), llama.SetSeed(seed))
			if err != nil {
				panic(err)
			}

			// Embeddings表示（オプション）
			if false { // デバッグ用フラグ
				embeds, err := l.Embeddings(text)
				if err != nil {
					fmt.Printf("Embeddings: error %s \n", err.Error())
				} else {
					fmt.Printf("Embeddings (first 5): %v...\n", embeds[:5])
				}
			}

			fmt.Printf("\n\n")
		}
	} else {
		// 非インタラクティブモード（C2シナリオ用）
		// 環境変数やコマンドライン引数からプロンプトを取得
		prompt := flags.Arg(0)
		if prompt == "" {
			prompt = os.Getenv("LLAMA_PROMPT")
		}
		if prompt == "" {
			prompt = "Hello, how can I help you today?"
		}

		fmt.Printf("Prompt: %s\n", prompt)
		fmt.Printf("Response: ")

		_, err := l.Predict(prompt, llama.SetTokenCallback(func(token string) bool {
			fmt.Print(token)
			return true
		}), llama.SetTokens(tokens), llama.SetThreads(threads), llama.SetTopK(90), llama.SetTopP(0.86), llama.SetSeed(seed))

		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}
}

// readMultiLineInput reads input until an empty line is entered.
func readMultiLineInput(reader *bufio.Reader) string {
	var lines []string
	fmt.Print(">>> ")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				os.Exit(0)
			}
			fmt.Printf("Reading the prompt failed: %s", err)
			os.Exit(1)
		}

		if len(strings.TrimSpace(line)) == 0 {
			break
		}

		lines = append(lines, line)
	}

	text := strings.Join(lines, "")
	fmt.Println("Sending", text)
	return text
}
