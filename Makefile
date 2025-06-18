.PHONY: test clean

INCLUDE_PATH := $(abspath ./)
LIBRARY_PATH := $(abspath ./)

ifndef UNAME_S
UNAME_S := $(shell uname -s)
endif

ifndef UNAME_P
UNAME_P := $(shell uname -p)
endif

ifndef UNAME_M
UNAME_M := $(shell uname -m)
endif

# Detect Windows (including MinGW/MSYS) early so we can use it for compiler defaults
ifneq ($(findstring _NT,$(UNAME_S)),)
	IS_WINDOWS := 1
endif
ifneq ($(findstring MINGW,$(UNAME_S)),)
	IS_WINDOWS := 1
endif
ifneq ($(findstring MSYS,$(UNAME_S)),)
	IS_WINDOWS := 1
endif

# Set default compilers if not defined
ifndef CC
	ifdef IS_WINDOWS
		CC = gcc
	else
		CC = cc
	endif
endif

ifndef CXX
	CXX = g++
endif

CCV := $(shell $(CC) --version | head -n 1)
CXXV := $(shell $(CXX) --version | head -n 1)

# Mac OS + Arm can report x86_64
# ref: https://github.com/ggerganov/whisper.cpp/issues/66#issuecomment-1282546789
ifeq ($(UNAME_S),Darwin)
	ifneq ($(UNAME_P),arm)
		SYSCTL_M := $(shell sysctl -n hw.optional.arm64 2>/dev/null)
		ifeq ($(SYSCTL_M),1)
			# UNAME_P := arm
			# UNAME_M := arm64
			warn := $(warning Your arch is announced as x86_64, but it seems to actually be ARM64. Not fixing that can lead to bad performance. For more info see: https://github.com/ggerganov/whisper.cpp/issues/66\#issuecomment-1282546789)
		endif
	endif
endif

#
# Compile flags
#

BUILD_TYPE?=
# keep standard at C11 and C++11
CFLAGS   = -I./llama.cpp -I. -O3 -DNDEBUG -std=c11 -fPIC
CXXFLAGS = -I./llama.cpp -I. -I./llama.cpp/common -I./common -O3 -DNDEBUG -std=c++11 -fPIC
LDFLAGS  =

# Ensure consistent ABI across all object files
ifdef IS_WINDOWS
	# Use standard flags for Windows builds
	CFLAGS = -I./llama.cpp -I. -O3 -DNDEBUG -std=c11 -fPIC
	CXXFLAGS = -I./llama.cpp -I. -I./llama.cpp/common -I./common -O3 -DNDEBUG -std=c++11 -fPIC
	# MinGW requires static linking of libstdc++ to avoid symbol issues
	LDFLAGS = -static-libgcc -static-libstdc++ -Wl,--whole-archive -lpthread -Wl,--no-whole-archive
	# Add to CMAKE_ARGS to ensure CMake uses the same flags
	# Also set CMAKE_CXX_STANDARD to force C++11 standard
	CMAKE_ARGS += -DCMAKE_C_FLAGS="-O3 -DNDEBUG -std=c11 -fPIC" \
	              -DCMAKE_CXX_FLAGS="-O3 -DNDEBUG -std=c++11 -fPIC" \
	              -DCMAKE_CXX_STANDARD=11 \
	              -DCMAKE_CXX_STANDARD_REQUIRED=ON \
	              -DCMAKE_EXE_LINKER_FLAGS="-static-libgcc -static-libstdc++" \
	              -DCMAKE_SHARED_LINKER_FLAGS="-static-libgcc -static-libstdc++"
endif

# warnings
CFLAGS   += -Wall -Wextra -Wpedantic -Wcast-qual -Wdouble-promotion -Wshadow -Wstrict-prototypes -Wpointer-arith -Wno-unused-function
CXXFLAGS += -Wall -Wextra -Wpedantic -Wcast-qual -Wno-unused-function

# OS specific
# TODO: support Windows

# Set object file extension based on platform
ifdef IS_WINDOWS
	OBJ_EXT := .o
	EXE_EXT := .exe
	# MinGW-specific flags
	LDFLAGS += -static-libgcc -static-libstdc++
	# Ensure proper linking of C++ standard library components
	LDFLAGS += -lstdc++ -lm
	# Additional flags to handle codecvt issues
	LDFLAGS += -Wl,--whole-archive -lpthread -Wl,--no-whole-archive
	# Define to avoid codecvt usage in MinGW
	CXXFLAGS += -D__MINGW32__ -D_WIN32_WINNT=0x0601
	WINDOWS_FLAGS_SET := 1
	export WINDOWS_FLAGS_SET
else
	OBJ_EXT := .o
	EXE_EXT :=
endif

ifeq ($(UNAME_S),Linux)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif
ifeq ($(UNAME_S),Darwin)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif
ifeq ($(UNAME_S),FreeBSD)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif
ifeq ($(UNAME_S),NetBSD)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif
ifeq ($(UNAME_S),OpenBSD)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif
ifeq ($(UNAME_S),Haiku)
	CFLAGS   += -pthread
	CXXFLAGS += -pthread
endif

# GPGPU specific
GGML_CUDA_OBJ_PATH=CMakeFiles/ggml.dir/ggml-cuda.cu.o


# Architecture specific
# TODO: probably these flags need to be tweaked on some architectures
#       feel free to update the Makefile for your architecture and send a pull request or issue
ifeq ($(UNAME_M),$(filter $(UNAME_M),x86_64 i686))
	# Use all CPU extensions that are available:
	CFLAGS += -march=native -mtune=native
endif
ifneq ($(filter ppc64%,$(UNAME_M)),)
	POWER9_M := $(shell grep "POWER9" /proc/cpuinfo)
	ifneq (,$(findstring POWER9,$(POWER9_M)))
		CFLAGS += -mcpu=power9
		CXXFLAGS += -mcpu=power9
	endif
	# Require c++23's std::byteswap for big-endian support.
	ifeq ($(UNAME_M),ppc64)
		CXXFLAGS += -std=c++23 -DGGML_BIG_ENDIAN
	endif
endif
ifndef LLAMA_NO_ACCELERATE
	# Mac M1 - include Accelerate framework.
	# `-framework Accelerate` works on Mac Intel as well, with negliable performance boost (as of the predict time).
	ifeq ($(UNAME_S),Darwin)
		CFLAGS  += -DGGML_USE_ACCELERATE
		LDFLAGS += -framework Accelerate
	endif
endif
ifdef LLAMA_OPENBLAS
	CFLAGS  += -DGGML_USE_OPENBLAS -I/usr/local/include/openblas
	LDFLAGS += -lopenblas
endif
ifdef LLAMA_GPROF
	CFLAGS   += -pg
	CXXFLAGS += -pg
endif
ifneq ($(filter aarch64%,$(UNAME_M)),)
	CFLAGS += -mcpu=native
	CXXFLAGS += -mcpu=native
endif
ifneq ($(filter armv6%,$(UNAME_M)),)
	# Raspberry Pi 1, 2, 3
	CFLAGS += -mfpu=neon-fp-armv8 -mfp16-format=ieee -mno-unaligned-access
endif
ifneq ($(filter armv7%,$(UNAME_M)),)
	# Raspberry Pi 4
	CFLAGS += -mfpu=neon-fp-armv8 -mfp16-format=ieee -mno-unaligned-access -funsafe-math-optimizations
endif
ifneq ($(filter armv8%,$(UNAME_M)),)
	# Raspberry Pi 4
	CFLAGS += -mfp16-format=ieee -mno-unaligned-access
endif

ifeq ($(BUILD_TYPE),openblas)
	EXTRA_LIBS=
	CMAKE_ARGS+=-DLLAMA_BLAS=ON -DLLAMA_BLAS_VENDOR=OpenBLAS -DBLAS_INCLUDE_DIRS=/usr/include/openblas
endif

ifeq ($(BUILD_TYPE),blis)
	EXTRA_LIBS=
	CMAKE_ARGS+=-DLLAMA_BLAS=ON -DLLAMA_BLAS_VENDOR=FLAME
endif

ifeq ($(BUILD_TYPE),cublas)
	EXTRA_LIBS=
	CMAKE_ARGS+=-DLLAMA_CUBLAS=ON
	EXTRA_TARGETS+=llama.cpp/ggml-cuda.o
endif

ifeq ($(BUILD_TYPE),hipblas)
	ROCM_HOME ?= "/opt/rocm"
	CXX="$(ROCM_HOME)"/llvm/bin/clang++
	CC="$(ROCM_HOME)"/llvm/bin/clang
	EXTRA_LIBS=
	GPU_TARGETS ?= gfx900,gfx90a,gfx1030,gfx1031,gfx1100
	AMDGPU_TARGETS ?= "$(GPU_TARGETS)"
	CMAKE_ARGS+=-DLLAMA_HIPBLAS=ON -DAMDGPU_TARGETS="$(AMDGPU_TARGETS)" -DGPU_TARGETS="$(GPU_TARGETS)"
	EXTRA_TARGETS+=llama.cpp/ggml-cuda.o
	GGML_CUDA_OBJ_PATH=CMakeFiles/ggml-rocm.dir/ggml-cuda.cu.o
endif

ifeq ($(BUILD_TYPE),clblas)
	EXTRA_LIBS=
	CMAKE_ARGS+=-DLLAMA_CLBLAST=ON
	EXTRA_TARGETS+=llama.cpp/ggml-opencl.o
endif

ifeq ($(BUILD_TYPE),metal)
	EXTRA_LIBS=
	CGO_LDFLAGS+="-framework Accelerate -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders"
	CMAKE_ARGS+=-DLLAMA_METAL=ON
	EXTRA_TARGETS+=llama.cpp/ggml-metal.o
endif

ifdef CLBLAST_DIR
	CMAKE_ARGS+=-DCLBlast_dir=$(CLBLAST_DIR)
endif

# TODO: support Windows
ifeq ($(GPU_TESTS),true)
	CGO_LDFLAGS="-lcublas -lcudart -L/usr/local/cuda/lib64/"
	TEST_LABEL=gpu
else
	TEST_LABEL=!gpu
endif

#
# Print build information
#

$(info I llama.cpp build info: )
$(info I UNAME_S:  $(UNAME_S))
$(info I UNAME_P:  $(UNAME_P))
$(info I UNAME_M:  $(UNAME_M))
$(info I CFLAGS:   $(CFLAGS))
$(info I CXXFLAGS: $(CXXFLAGS))
$(info I CGO_LDFLAGS:  $(CGO_LDFLAGS))
$(info I LDFLAGS:  $(LDFLAGS))
$(info I BUILD_TYPE:  $(BUILD_TYPE))
$(info I CMAKE_ARGS:  $(CMAKE_ARGS))
$(info I EXTRA_TARGETS:  $(EXTRA_TARGETS))
$(info I CC:       $(CCV))
$(info I CXX:      $(CXXV))
$(info )

# Use this if you want to set the default behavior

# Debug output for Windows builds
ifdef IS_WINDOWS
$(info I Windows build detected)
$(info I Final CFLAGS: $(CFLAGS))
$(info I Final CXXFLAGS: $(CXXFLAGS))
$(info I Final CMAKE_ARGS: $(CMAKE_ARGS))
endif

llama.cpp/grammar-parser.o: llama.cpp/ggml.o
ifdef IS_WINDOWS
	cd build && (cp -rf common/CMakeFiles/common.dir/grammar-parser.cpp.obj ../llama.cpp/grammar-parser.o 2>/dev/null || cp -rf common/CMakeFiles/common.dir/grammar-parser.cpp.o ../llama.cpp/grammar-parser.o)
else
	cd build && cp -rf common/CMakeFiles/common.dir/grammar-parser.cpp.o ../llama.cpp/grammar-parser.o
endif

llama.cpp/ggml-alloc.o: llama.cpp/ggml.o
ifdef IS_WINDOWS
	cd build && (cp -rf CMakeFiles/ggml.dir/ggml-alloc.c.obj ../llama.cpp/ggml-alloc.o 2>/dev/null || cp -rf CMakeFiles/ggml.dir/ggml-alloc.c.o ../llama.cpp/ggml-alloc.o)
else
	cd build && cp -rf CMakeFiles/ggml.dir/ggml-alloc.c.o ../llama.cpp/ggml-alloc.o
endif

llama.cpp/ggml.o: prepare
	mkdir -p build
ifdef IS_WINDOWS
	# Export flags to ensure CMake picks them up
	cd build && \
		export CFLAGS="$(CFLAGS)" && \
		export CXXFLAGS="$(CXXFLAGS)" && \
		CC="$(CC)" CXX="$(CXX)" cmake -G "MinGW Makefiles" ../llama.cpp \
			-DCMAKE_C_FLAGS="$(CFLAGS)" \
			-DCMAKE_CXX_FLAGS="$(CXXFLAGS)" \
			-DCMAKE_C_FLAGS_RELEASE="$(CFLAGS)" \
			-DCMAKE_CXX_FLAGS_RELEASE="$(CXXFLAGS)" \
			$(CMAKE_ARGS) && \
		VERBOSE=1 cmake --build . --config Release && \
		(cp -rf CMakeFiles/ggml.dir/ggml.c.obj ../llama.cpp/ggml.o 2>/dev/null || cp -rf CMakeFiles/ggml.dir/ggml.c.o ../llama.cpp/ggml.o)
else
	cd build && CC="$(CC)" CXX="$(CXX)" cmake ../llama.cpp $(CMAKE_ARGS) && VERBOSE=1 cmake --build . --config Release && cp -rf CMakeFiles/ggml.dir/ggml.c.o ../llama.cpp/ggml.o
endif

llama.cpp/ggml-cuda.o: llama.cpp/ggml.o
	cd build && cp -rf "$(GGML_CUDA_OBJ_PATH)" ../llama.cpp/ggml-cuda.o

llama.cpp/ggml-opencl.o: llama.cpp/ggml.o
	cd build && cp -rf CMakeFiles/ggml.dir/ggml-opencl.cpp.o ../llama.cpp/ggml-opencl.o

llama.cpp/ggml-metal.o: llama.cpp/ggml.o
	cd build && cp -rf CMakeFiles/ggml.dir/ggml-metal.m.o ../llama.cpp/ggml-metal.o

llama.cpp/k_quants.o: llama.cpp/ggml.o
ifdef IS_WINDOWS
	cd build && (cp -rf CMakeFiles/ggml.dir/k_quants.c.obj ../llama.cpp/k_quants.o 2>/dev/null || cp -rf CMakeFiles/ggml.dir/k_quants.c.o ../llama.cpp/k_quants.o)
else
	cd build && cp -rf CMakeFiles/ggml.dir/k_quants.c.o ../llama.cpp/k_quants.o
endif

llama.cpp/llama.o: llama.cpp/ggml.o
ifdef IS_WINDOWS
	cd build && (cp -rf CMakeFiles/llama.dir/llama.cpp.obj ../llama.cpp/llama.o 2>/dev/null || cp -rf CMakeFiles/llama.dir/llama.cpp.o ../llama.cpp/llama.o)
else
	cd build && cp -rf CMakeFiles/llama.dir/llama.cpp.o ../llama.cpp/llama.o
endif

llama.cpp/common.o: llama.cpp/ggml.o
ifdef IS_WINDOWS
	cd build && (cp -rf common/CMakeFiles/common.dir/common.cpp.obj ../llama.cpp/common.o 2>/dev/null || cp -rf common/CMakeFiles/common.dir/common.cpp.o ../llama.cpp/common.o)
else
	cd build && cp -rf common/CMakeFiles/common.dir/common.cpp.o ../llama.cpp/common.o
endif

binding.o: prepare
ifdef IS_WINDOWS
	@echo "Compiling binding.cpp with: $(CXX) $(CXXFLAGS)"
endif
	$(CXX) $(CXXFLAGS) -I./llama.cpp -I./llama.cpp/common binding.cpp -o binding.o -c

llama_data_source.o: prepare
ifdef IS_WINDOWS
	@echo "Compiling llama_data_source.cpp with: $(CXX) $(CXXFLAGS)"
endif
	$(CXX) $(CXXFLAGS) -I./llama.cpp -I./llama.cpp/common llama_data_source.cpp -o llama_data_source.o -c

## https://github.com/ggerganov/llama.cpp/pull/1902
prepare:
	cd llama.cpp && patch -p1 < ../patches/1902-cuda.patch
	cd llama.cpp && patch -p1 < ../patches/memory-loading.patch
ifdef IS_WINDOWS
	cd llama.cpp && patch -p1 < ../patches/mingw-codecvt-fix.patch
	cd llama.cpp && patch -p1 < ../patches/mingw-win32-memory-range.patch
	cd llama.cpp && patch -p1 < ../patches/mingw-seekp-seekg-fix.patch
endif
	touch $@

libbinding.a: llama.cpp/ggml.o llama.cpp/k_quants.o llama.cpp/ggml-alloc.o llama.cpp/common.o llama.cpp/grammar-parser.o llama.cpp/llama.o binding.o llama_data_source.o $(EXTRA_TARGETS)
ifdef IS_WINDOWS
	# Use x86_64-w64-mingw32-ar if available for better cross-compilation compatibility
	@if command -v x86_64-w64-mingw32-ar >/dev/null 2>&1; then \
		echo "Using x86_64-w64-mingw32-ar"; \
		x86_64-w64-mingw32-ar rcs libbinding.a llama.cpp/ggml.o llama.cpp/k_quants.o llama.cpp/ggml-alloc.o llama.cpp/common.o llama.cpp/grammar-parser.o llama.cpp/llama.o binding.o llama_data_source.o $(EXTRA_TARGETS); \
	else \
		ar rcs libbinding.a llama.cpp/ggml.o llama.cpp/k_quants.o llama.cpp/ggml-alloc.o llama.cpp/common.o llama.cpp/grammar-parser.o llama.cpp/llama.o binding.o llama_data_source.o $(EXTRA_TARGETS); \
	fi
else
	ar rcs libbinding.a llama.cpp/ggml.o llama.cpp/k_quants.o llama.cpp/ggml-alloc.o llama.cpp/common.o llama.cpp/grammar-parser.o llama.cpp/llama.o binding.o llama_data_source.o $(EXTRA_TARGETS)
endif

clean:
	rm -rf *.o
	rm -rf *.a
	$(MAKE) -C llama.cpp clean
	rm -rf build

# Use a smaller model for faster testing (TinyLlama 1.1B instead of CodeLlama 7B)
ggllm-test-model.bin:
ifdef IS_WINDOWS
	@echo Downloading test model...
	@curl -L --progress-bar -o ggllm-test-model.bin https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf || \
	powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ProgressPreference = 'SilentlyContinue'; Invoke-WebRequest -Uri 'https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf' -OutFile 'ggllm-test-model.bin'"
else
	wget -q https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf -O ggllm-test-model.bin
endif

test: ggllm-test-model.bin libbinding.a
	C_INCLUDE_PATH=${INCLUDE_PATH} CGO_LDFLAGS=${CGO_LDFLAGS} LIBRARY_PATH=${LIBRARY_PATH} TEST_MODEL=ggllm-test-model.bin go run github.com/onsi/ginkgo/v2/ginkgo --label-filter="$(TEST_LABEL)" --flake-attempts 5 --skip-package=examples -v -r ./...