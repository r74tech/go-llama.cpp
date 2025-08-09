package llama

// #cgo CXXFLAGS: -I${SRCDIR}/llama.cpp/common -I${SRCDIR}/llama.cpp
// #cgo LDFLAGS: -L${SRCDIR}/ -lbinding -lm -lstdc++
// #cgo darwin LDFLAGS: -framework Accelerate -framework Foundation -framework Metal -framework MetalKit
// #cgo darwin CXXFLAGS: -std=c++11
// #cgo windows LDFLAGS: -static -static-libgcc -static-libstdc++ -lpthread
// #include "binding.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

type LLama struct {
	state       unsafe.Pointer
	embeddings  bool
	contextSize int
	// Keep a reference to the model data to prevent GC
	modelData []byte
	// Keep the model bytes pinned for the lifetime of the model (Go 1.21+)
	pin runtime.Pinner
	// Mutex to protect concurrent predict calls
	predictMu sync.Mutex
}

func New(model string, opts ...ModelOption) (*LLama, error) {
	mo := NewModelOptions(opts...)
	modelPath := C.CString(model)
	defer C.free(unsafe.Pointer(modelPath))
	loraBase := C.CString(mo.LoraBase)
	defer C.free(unsafe.Pointer(loraBase))
	loraAdapter := C.CString(mo.LoraAdapter)
	defer C.free(unsafe.Pointer(loraAdapter))

	MulMatQ := true

	if mo.MulMatQ != nil {
		MulMatQ = *mo.MulMatQ
	}

	result := C.load_model(modelPath,
		C.int(mo.ContextSize), C.int(mo.Seed),
		C.bool(mo.F16Memory), C.bool(mo.MLock), C.bool(mo.Embeddings), C.bool(mo.MMap), C.bool(mo.LowVRAM),
		C.int(mo.NGPULayers), C.int(mo.NBatch), C.CString(mo.MainGPU), C.CString(mo.TensorSplit), C.bool(mo.NUMA),
		C.float(mo.FreqRopeBase), C.float(mo.FreqRopeScale),
		C.bool(MulMatQ), loraAdapter, loraBase, C.bool(mo.Perplexity),
	)

	if result == nil {
		return nil, fmt.Errorf("failed loading model from %s - model file may not exist or is invalid", model)
	}

	ll := &LLama{state: result, contextSize: mo.ContextSize, embeddings: mo.Embeddings}
	return ll, nil
}

func NewFromMemory(modelData []byte, opts ...ModelOption) (*LLama, error) {
	mo := NewModelOptions(opts...)
	loraBase := C.CString(mo.LoraBase)
	defer C.free(unsafe.Pointer(loraBase))
	loraAdapter := C.CString(mo.LoraAdapter)
	defer C.free(unsafe.Pointer(loraAdapter))

	MulMatQ := true

	if mo.MulMatQ != nil {
		MulMatQ = *mo.MulMatQ
	}

	if len(modelData) == 0 {
		return nil, fmt.Errorf("model data is empty")
	}

	// Allocate C strings up-front and free them after the call
	mainGPU := C.CString(mo.MainGPU)
	defer C.free(unsafe.Pointer(mainGPU))
	tensorSplit := C.CString(mo.TensorSplit)
	defer C.free(unsafe.Pointer(tensorSplit))

	// Zero-copy: pass a pointer into the Go slice to C and pin it to make it
	// non-movable by GC while C code may access it.
	dataPtr := unsafe.Pointer(&modelData[0])
	dataSize := C.size_t(len(modelData))
	var pinner runtime.Pinner
	pinner.Pin(&modelData[0])

	// Debug
	if os.Getenv("LLAMA_DEBUG") != "" {
		fmt.Printf("NewFromMemory: using Go buffer %d bytes at %p (zero-copy)\n", dataSize, dataPtr)
	}

	result := C.load_model_from_memory(dataPtr, dataSize,
		C.int(mo.ContextSize), C.int(mo.Seed),
		C.bool(mo.F16Memory), C.bool(mo.MLock), C.bool(mo.Embeddings), C.bool(mo.MMap), C.bool(mo.LowVRAM),
		C.int(mo.NGPULayers), C.int(mo.NBatch), mainGPU, tensorSplit, C.bool(mo.NUMA),
		C.float(mo.FreqRopeBase), C.float(mo.FreqRopeScale),
		C.bool(MulMatQ), loraAdapter, loraBase, C.bool(mo.Perplexity),
	)

	if result == nil {
		// Unpin on failure
		pinner.Unpin()
		return nil, fmt.Errorf("failed loading model from memory")
	}

	ll := &LLama{
		state:       result,
		contextSize: mo.ContextSize,
		embeddings:  mo.Embeddings,
		modelData:   modelData, // Keep reference to prevent GC
	}
	// Transfer the pinner to the struct to keep it pinned for the lifetime of l
	ll.pin = pinner
	// Pin the underlying array for the lifetime of the model to ensure C does
	// not observe the memory moved or freed by the GC.
	// Note: already pinned above before calling into C; keep it pinned until Free.
	return ll, nil
}

// LoadSelfContainedModel loads a model that has been appended to the current binary
// It detects self-contained models and uses zero-copy mmap loading
func LoadSelfContainedModel(opts ...ModelOption) (*LLama, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Open the executable for reading
	file, err := os.Open(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}
	fileSize := info.Size()

	// Read the last 8 bytes to get the model size (uint64 in little-endian)
	if fileSize < 8 {
		return nil, fmt.Errorf("executable too small to contain self-contained model")
	}

	_, err = file.Seek(-8, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to model size marker: %w", err)
	}

	var modelSize uint64
	err = binary.Read(file, binary.LittleEndian, &modelSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read model size: %w", err)
	}

	// Validate model size
	if modelSize == 0 || modelSize > uint64(fileSize-8) {
		return nil, fmt.Errorf("invalid model size: %d (file size: %d)", modelSize, fileSize)
	}

	fmt.Printf("Found self-contained model of size %d MB, mapping into memory...\n", modelSize/(1024*1024))

	// Calculate the offset where the model starts
	modelOffset := fileSize - int64(modelSize) - 8

	// Memory map the model region using platform-specific implementation
	fd := int(file.Fd())

	addr, _, err := mmapModel(fd, modelOffset, int(modelSize))
	if err != nil {
		// Fallback to standard memory loading if mmap fails
		fmt.Printf("mmap failed (%v), falling back to standard memory loading\n", err)

		// Seek to model start
		_, err = file.Seek(modelOffset, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to model start: %w", err)
		}

		// Read model data
		modelData := make([]byte, modelSize)
		_, err = io.ReadFull(file, modelData)
		if err != nil {
			return nil, fmt.Errorf("failed to read model data: %w", err)
		}

		fmt.Printf("Successfully loaded self-contained model into memory\n")
		return NewFromMemory(modelData, opts...)
	}

	fmt.Printf("Successfully mapped self-contained model at address %p\n", unsafe.Pointer(addr))

	// Use the zero-copy mmap loader
	return NewFromMMap(addr, int(modelSize), opts...)
}

// NewFromMMap creates a new LLama model from a memory-mapped region (zero-copy)
// The memory region must remain valid for the lifetime of the model
func NewFromMMap(addr uintptr, size int, opts ...ModelOption) (*LLama, error) {
	mo := NewModelOptions(opts...)

	// Force mmap mode for zero-copy
	mo.MMap = true

	loraBase := C.CString(mo.LoraBase)
	defer C.free(unsafe.Pointer(loraBase))
	loraAdapter := C.CString(mo.LoraAdapter)
	defer C.free(unsafe.Pointer(loraAdapter))

	MulMatQ := true
	if mo.MulMatQ != nil {
		MulMatQ = *mo.MulMatQ
	}

	if size == 0 {
		return nil, fmt.Errorf("mmap size is zero")
	}

	// Allocate C strings up-front and free them after the call
	mainGPU := C.CString(mo.MainGPU)
	defer C.free(unsafe.Pointer(mainGPU))
	tensorSplit := C.CString(mo.TensorSplit)
	defer C.free(unsafe.Pointer(tensorSplit))

	// Convert address to unsafe.Pointer
	dataPtr := unsafe.Pointer(addr)
	dataSize := C.size_t(size)

	// Debug
	if os.Getenv("LLAMA_DEBUG") != "" {
		fmt.Printf("NewFromMMap: using mmap'd memory %d bytes at %p (zero-copy)\n", dataSize, dataPtr)
	}

	result := C.load_model_from_mmap(dataPtr, dataSize,
		C.int(mo.ContextSize), C.int(mo.Seed),
		C.bool(mo.F16Memory), C.bool(mo.MLock), C.bool(mo.Embeddings), C.bool(true), C.bool(mo.LowVRAM),
		C.int(mo.NGPULayers), C.int(mo.NBatch), mainGPU, tensorSplit, C.bool(mo.NUMA),
		C.float(mo.FreqRopeBase), C.float(mo.FreqRopeScale),
		C.bool(MulMatQ), loraAdapter, loraBase, C.bool(mo.Perplexity),
	)

	if result == nil {
		return nil, fmt.Errorf("failed loading model from mmap")
	}

	ll := &LLama{
		state:       result,
		contextSize: mo.ContextSize,
		embeddings:  mo.Embeddings,
		// No modelData or pin for mmap - memory is externally managed
	}

	return ll, nil
}

func (l *LLama) Free() {
	C.llama_binding_free_model(l.state)
	// Unpin after the model is freed on the C side
	if len(l.modelData) > 0 {
		l.pin.Unpin()
	}
}

func (l *LLama) LoadState(state string) error {
	d := C.CString(state)
	w := C.CString("rb")
	result := C.load_state(l.state, d, w)

	defer C.free(unsafe.Pointer(d)) // free allocated C string
	defer C.free(unsafe.Pointer(w)) // free allocated C string

	if result != 0 {
		return fmt.Errorf("error while loading state")
	}

	return nil
}

func (l *LLama) SaveState(dst string) error {
	d := C.CString(dst)
	w := C.CString("wb")

	C.save_state(l.state, d, w)

	defer C.free(unsafe.Pointer(d)) // free allocated C string
	defer C.free(unsafe.Pointer(w)) // free allocated C string

	_, err := os.Stat(dst)
	return err
}

// Token Embeddings
func (l *LLama) TokenEmbeddings(tokens []int, opts ...PredictOption) ([]float32, error) {
	if !l.embeddings {
		return []float32{}, fmt.Errorf("model loaded without embeddings")
	}

	// Protect against concurrent token embeddings calls
	l.predictMu.Lock()
	defer l.predictMu.Unlock()

	po := NewPredictOptions(opts...)

	outSize := po.Tokens
	if po.Tokens == 0 {
		outSize = 9999999
	}

	floats := make([]float32, outSize)

	myArray := (*C.int)(C.malloc(C.size_t(len(tokens)) * C.sizeof_int))

	// Copy the values from the Go slice to the C array
	for i, v := range tokens {
		(*[1<<31 - 1]int32)(unsafe.Pointer(myArray))[i] = int32(v)
	}
	// void* llama_allocate_params(const char *prompt, int seed, int threads, int tokens,
	// int top_k, float top_p, float temp, float repeat_penalty,
	// int repeat_last_n, bool ignore_eos, bool memory_f16,
	// int n_batch, int n_keep, const char** antiprompt, int antiprompt_count,
	// float tfs_z, float typical_p, float frequency_penalty, float presence_penalty, int mirostat, float mirostat_eta, float mirostat_tau, bool penalize_nl, const char *logit_bias, const char *session_file, bool prompt_cache_all, bool mlock, bool mmap, const char *maingpu, const char *tensorsplit , bool prompt_cache_ro,
	// float rope_freq_base, float rope_freq_scale, float negative_prompt_scale, const char* negative_prompt
	// );
	params := C.llama_allocate_params(C.CString(""), C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), nil, C.int(0),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), C.CString(po.LogitBias),
		C.CString(po.PathPromptCache), C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		C.CString(po.MainGPU), C.CString(po.TensorSplit),
		C.bool(po.PromptCacheRO),
		C.CString(po.Grammar),
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), C.CString(po.NegativePrompt),
		C.int(po.NDraft),
	)
	ret := C.get_token_embeddings(params, l.state, myArray, C.int(len(tokens)), (*C.float)(&floats[0]))
	if ret != 0 {
		return floats, fmt.Errorf("embedding inference failed")
	}
	return floats, nil
}

// Embeddings
func (l *LLama) Embeddings(text string, opts ...PredictOption) ([]float32, error) {
	if !l.embeddings {
		return []float32{}, fmt.Errorf("model loaded without embeddings")
	}

	// Protect against concurrent embeddings calls
	l.predictMu.Lock()
	defer l.predictMu.Unlock()

	po := NewPredictOptions(opts...)

	input := C.CString(text)
	if po.Tokens == 0 {
		po.Tokens = 99999999
	}
	floats := make([]float32, po.Tokens)
	reverseCount := len(po.StopPrompts)
	reversePrompt := make([]*C.char, reverseCount)
	var pass **C.char
	for i, s := range po.StopPrompts {
		cs := C.CString(s)
		reversePrompt[i] = cs
		pass = &reversePrompt[0]
	}

	params := C.llama_allocate_params(input, C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), pass, C.int(reverseCount),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), C.CString(po.LogitBias),
		C.CString(po.PathPromptCache), C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		C.CString(po.MainGPU), C.CString(po.TensorSplit),
		C.bool(po.PromptCacheRO),
		C.CString(po.Grammar),
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), C.CString(po.NegativePrompt),
		C.int(po.NDraft),
	)

	ret := C.get_embeddings(params, l.state, (*C.float)(&floats[0]))
	if ret != 0 {
		return floats, fmt.Errorf("embedding inference failed")
	}

	return floats, nil
}

func (l *LLama) Eval(text string, opts ...PredictOption) error {
	// Protect against concurrent eval calls
	l.predictMu.Lock()
	defer l.predictMu.Unlock()

	po := NewPredictOptions(opts...)

	input := C.CString(text)
	if po.Tokens == 0 {
		po.Tokens = 99999999
	}

	reverseCount := len(po.StopPrompts)
	reversePrompt := make([]*C.char, reverseCount)
	var pass **C.char
	for i, s := range po.StopPrompts {
		cs := C.CString(s)
		reversePrompt[i] = cs
		pass = &reversePrompt[0]
	}

	params := C.llama_allocate_params(input, C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), pass, C.int(reverseCount),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), C.CString(po.LogitBias),
		C.CString(po.PathPromptCache), C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		C.CString(po.MainGPU), C.CString(po.TensorSplit),
		C.bool(po.PromptCacheRO),
		C.CString(po.Grammar),
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), C.CString(po.NegativePrompt),
		C.int(po.NDraft),
	)
	ret := C.eval(params, l.state, input)
	if ret != 0 {
		return fmt.Errorf("inference failed")
	}

	C.llama_free_params(params)

	return nil
}

func (l *LLama) SpeculativeSampling(ll *LLama, text string, opts ...PredictOption) (string, error) {
	// Protect against concurrent predictions
	l.predictMu.Lock()
	defer l.predictMu.Unlock()
	if ll != l {
		ll.predictMu.Lock()
		defer ll.predictMu.Unlock()
	}

	po := NewPredictOptions(opts...)

	if po.TokenCallback != nil {
		setCallback(l.state, po.TokenCallback)
	}

	input := C.CString(text)
	if po.Tokens == 0 {
		po.Tokens = 99999999
	}
	out := make([]byte, po.Tokens)

	reverseCount := len(po.StopPrompts)
	reversePrompt := make([]*C.char, reverseCount)
	var pass **C.char
	for i, s := range po.StopPrompts {
		cs := C.CString(s)
		reversePrompt[i] = cs
		pass = &reversePrompt[0]
	}

	params := C.llama_allocate_params(input, C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), pass, C.int(reverseCount),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), C.CString(po.LogitBias),
		C.CString(po.PathPromptCache), C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		C.CString(po.MainGPU), C.CString(po.TensorSplit),
		C.bool(po.PromptCacheRO),
		C.CString(po.Grammar),
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), C.CString(po.NegativePrompt),
		C.int(po.NDraft),
	)
	ret := C.speculative_sampling(params, l.state, ll.state, (*C.char)(unsafe.Pointer(&out[0])), C.size_t(len(out)), C.bool(po.DebugMode))
	if ret != 0 {
		return "", fmt.Errorf("inference failed")
	}
	res := C.GoString((*C.char)(unsafe.Pointer(&out[0])))

	res = strings.TrimPrefix(res, " ")
	res = strings.TrimPrefix(res, text)
	res = strings.TrimPrefix(res, "\n")

	for _, s := range po.StopPrompts {
		res = strings.TrimRight(res, s)
	}

	C.llama_free_params(params)

	if po.TokenCallback != nil {
		setCallback(l.state, nil)
	}

	return res, nil
}

func (l *LLama) Predict(text string, opts ...PredictOption) (string, error) {
	// Protect against concurrent predictions
	l.predictMu.Lock()
	defer l.predictMu.Unlock()

	po := NewPredictOptions(opts...)

	if po.TokenCallback != nil {
		setCallback(l.state, po.TokenCallback)
	}

	input := C.CString(text)
	defer C.free(unsafe.Pointer(input))
	if po.Tokens == 0 {
		po.Tokens = 99999999
	}

	// Allocate C memory for output to avoid Go GC issues
	outSize := C.size_t(po.Tokens)
	outPtr := C.malloc(outSize)
	if outPtr == nil {
		return "", fmt.Errorf("failed to allocate memory for output")
	}
	defer C.free(outPtr)
	// Clear the allocated memory
	C.memset(outPtr, 0, outSize)

	reverseCount := len(po.StopPrompts)
	reversePrompt := make([]*C.char, reverseCount)
	var pass **C.char
	for i, s := range po.StopPrompts {
		cs := C.CString(s)
		reversePrompt[i] = cs
		defer C.free(unsafe.Pointer(cs))
		pass = &reversePrompt[0]
	}

	// Allocate C strings and ensure they are freed
	logitBias := C.CString(po.LogitBias)
	defer C.free(unsafe.Pointer(logitBias))
	pathPromptCache := C.CString(po.PathPromptCache)
	defer C.free(unsafe.Pointer(pathPromptCache))
	mainGPU := C.CString(po.MainGPU)
	defer C.free(unsafe.Pointer(mainGPU))
	tensorSplit := C.CString(po.TensorSplit)
	defer C.free(unsafe.Pointer(tensorSplit))
	grammar := C.CString(po.Grammar)
	defer C.free(unsafe.Pointer(grammar))
	negativePrompt := C.CString(po.NegativePrompt)
	defer C.free(unsafe.Pointer(negativePrompt))

	params := C.llama_allocate_params(input, C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), pass, C.int(reverseCount),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), logitBias,
		pathPromptCache, C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		mainGPU, tensorSplit,
		C.bool(po.PromptCacheRO),
		grammar,
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), negativePrompt,
		C.int(po.NDraft),
	)
	defer C.llama_free_params(params)

	ret := C.llama_predict(params, l.state, (*C.char)(outPtr), outSize, C.bool(po.DebugMode))
	if ret != 0 {
		return "", fmt.Errorf("inference failed")
	}
	res := C.GoString((*C.char)(outPtr))

	res = strings.TrimPrefix(res, " ")
	res = strings.TrimPrefix(res, text)
	res = strings.TrimPrefix(res, "\n")

	for _, s := range po.StopPrompts {
		res = strings.TrimRight(res, s)
	}

	if po.TokenCallback != nil {
		setCallback(l.state, nil)
	}

	// Ensure the LLama struct doesn't get garbage collected while C code is using it
	runtime.KeepAlive(l)

	return res, nil
}

// tokenize has an interesting return property: negative lengths (potentially) have meaning.
// Therefore, return the length seperate from the slice and error - all three can be used together
func (l *LLama) TokenizeString(text string, opts ...PredictOption) (int32, []int32, error) {
	po := NewPredictOptions(opts...)

	input := C.CString(text)
	if po.Tokens == 0 {
		po.Tokens = 4096 // ???
	}
	out := make([]C.int, po.Tokens)

	var fakeDblPtr **C.char

	// copy pasted and modified minimally. Should I simplify down / do we need an "allocate defaults"
	params := C.llama_allocate_params(input, C.int(po.Seed), C.int(po.Threads), C.int(po.Tokens), C.int(po.TopK),
		C.float(po.TopP), C.float(po.Temperature), C.float(po.Penalty), C.int(po.Repeat),
		C.bool(po.IgnoreEOS), C.bool(po.F16KV),
		C.int(po.Batch), C.int(po.NKeep), fakeDblPtr, C.int(0),
		C.float(po.TailFreeSamplingZ), C.float(po.TypicalP), C.float(po.FrequencyPenalty), C.float(po.PresencePenalty),
		C.int(po.Mirostat), C.float(po.MirostatETA), C.float(po.MirostatTAU), C.bool(po.PenalizeNL), C.CString(po.LogitBias),
		C.CString(po.PathPromptCache), C.bool(po.PromptCacheAll), C.bool(po.MLock), C.bool(po.MMap),
		C.CString(po.MainGPU), C.CString(po.TensorSplit),
		C.bool(po.PromptCacheRO),
		C.CString(po.Grammar),
		C.float(po.RopeFreqBase), C.float(po.RopeFreqScale), C.float(po.NegativePromptScale), C.CString(po.NegativePrompt),
		C.int(po.NDraft),
	)

	tokRet := C.llama_tokenize_string(params, l.state, (*C.int)(unsafe.Pointer(&out[0]))) //, C.int(po.Tokens), true)

	if tokRet < 0 {
		return int32(tokRet), []int32{}, fmt.Errorf("llama_tokenize_string returned negative count %d", tokRet)
	}

	// TODO: Is this loop still required to unbox cgo to go?
	gTokRet := int32(tokRet)

	gLenOut := min(len(out), int(gTokRet))

	goSlice := make([]int32, gLenOut)
	for i := 0; i < gLenOut; i++ {
		goSlice[i] = int32(out[i])
	}

	return gTokRet, goSlice, nil
}

// CGo only allows us to use static calls from C to Go, we can't just dynamically pass in func's.
// This is the next best thing, we register the callbacks in this map and call tokenCallback from
// the C code. We also attach a finalizer to LLama, so it will unregister the callback when the
// garbage collection frees it.

// SetTokenCallback registers a callback for the individual tokens created when running Predict. It
// will be called once for each token. The callback shall return true as long as the model should
// continue predicting the next token. When the callback returns false the predictor will return.
// The tokens are just converted into Go strings, they are not trimmed or otherwise changed. Also
// the tokens may not be valid UTF-8.
// Pass in nil to remove a callback.
//
// It is save to call this method while a prediction is running.
func (l *LLama) SetTokenCallback(callback func(token string) bool) {
	setCallback(l.state, callback)
}

var (
	m         sync.RWMutex
	callbacks = map[uintptr]func(string) bool{}
)

//export tokenCallback
func tokenCallback(statePtr unsafe.Pointer, token *C.char) bool {
	m.RLock()
	defer m.RUnlock()

	if callback, ok := callbacks[uintptr(statePtr)]; ok {
		return callback(C.GoString(token))
	}

	return true
}

// setCallback can be used to register a token callback for LLama. Pass in a nil callback to
// remove the callback.
func setCallback(statePtr unsafe.Pointer, callback func(string) bool) {
	m.Lock()
	defer m.Unlock()

	if callback == nil {
		delete(callbacks, uintptr(statePtr))
	} else {
		callbacks[uintptr(statePtr)] = callback
	}
}
