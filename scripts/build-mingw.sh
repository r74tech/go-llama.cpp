#!/bin/bash
# Build script for MinGW-w64 cross-compilation
# This script ensures compatibility with Linux cross-compilation environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building go-llama.cpp with MinGW-w64 for Windows...${NC}"

# Detect if we're on Windows or cross-compiling from Linux
if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "mingw"* ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    echo -e "${YELLOW}Building on Windows with MinGW${NC}"
    IS_CROSS_COMPILE=0
    CC=gcc
    CXX=g++
    AR=ar
elif command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    echo -e "${YELLOW}Cross-compiling from Linux to Windows${NC}"
    IS_CROSS_COMPILE=1
    CC=x86_64-w64-mingw32-gcc
    CXX=x86_64-w64-mingw32-g++
    AR=x86_64-w64-mingw32-ar
else
    echo -e "${RED}ERROR: MinGW-w64 not found. Please install mingw-w64 package.${NC}"
    echo "On Ubuntu/Debian: sudo apt-get install mingw-w64"
    echo "On Fedora: sudo dnf install mingw64-gcc mingw64-gcc-c++"
    echo "On Arch: sudo pacman -S mingw-w64-gcc"
    exit 1
fi

# Export compiler settings
export CC=$CC
export CXX=$CXX
export AR=$AR

# Set compilation flags for MinGW compatibility
export CFLAGS="-O3 -DNDEBUG -std=c11 -fPIC"
export CXXFLAGS="-O3 -DNDEBUG -std=c++11 -fPIC"
export LDFLAGS="-static-libgcc -static-libstdc++ -lm"

# Additional flags for cross-compilation
if [ $IS_CROSS_COMPILE -eq 1 ]; then
    export CFLAGS="$CFLAGS -pthread"
    export CXXFLAGS="$CXXFLAGS -pthread"
fi

echo -e "${GREEN}Compiler configuration:${NC}"
echo "CC: $CC"
echo "CXX: $CXX"
echo "AR: $AR"
echo "CFLAGS: $CFLAGS"
echo "CXXFLAGS: $CXXFLAGS"
echo "LDFLAGS: $LDFLAGS"

# Clean previous builds
echo -e "${GREEN}Cleaning previous builds...${NC}"
make clean

# Build the library
echo -e "${GREEN}Building libbinding.a...${NC}"
make libbinding.a

# Check if build was successful
if [ -f "libbinding.a" ]; then
    echo -e "${GREEN}Build successful!${NC}"
    echo -e "${GREEN}Library information:${NC}"
    file libbinding.a
    
    # Check for C++11 ABI symbols
    echo -e "${GREEN}Checking for C++11 ABI symbols:${NC}"
    if [ $IS_CROSS_COMPILE -eq 1 ]; then
        x86_64-w64-mingw32-nm libbinding.a 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -10 || echo "No C++11 ABI symbols found"
    else
        nm libbinding.a 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -10 || echo "No C++11 ABI symbols found"
    fi
    
    # Check for llama symbols
    echo -e "${GREEN}Checking for key llama symbols:${NC}"
    if [ $IS_CROSS_COMPILE -eq 1 ]; then
        x86_64-w64-mingw32-nm libbinding.a 2>/dev/null | grep -E "llama_tokenize|llama_token_to_piece" | head -10
    else
        nm libbinding.a 2>/dev/null | grep -E "llama_tokenize|llama_token_to_piece" | head -10
    fi
    
    # Extract and check individual object files
    echo -e "${GREEN}Extracting and checking object files:${NC}"
    mkdir -p temp_check
    cd temp_check
    if [ $IS_CROSS_COMPILE -eq 1 ]; then
        x86_64-w64-mingw32-ar x ../libbinding.a
        echo "Checking binding.o for ABI symbols:"
        x86_64-w64-mingw32-nm binding.o 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -5 || echo "No C++11 ABI in binding.o"
        echo "Checking llama.o for ABI symbols:"
        x86_64-w64-mingw32-nm llama.o 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -5 || echo "No C++11 ABI in llama.o"
    else
        ar x ../libbinding.a
        echo "Checking binding.o for ABI symbols:"
        nm binding.o 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -5 || echo "No C++11 ABI in binding.o"
        echo "Checking llama.o for ABI symbols:"
        nm llama.o 2>/dev/null | grep -E "(cxx11|__cxx11)" | head -5 || echo "No C++11 ABI in llama.o"
    fi
    cd ..
    rm -rf temp_check
    
    echo -e "${GREEN}Library size:${NC}"
    ls -lh libbinding.a
else
    echo -e "${RED}Build failed! libbinding.a not found.${NC}"
    exit 1
fi

echo -e "${GREEN}Done! The library is ready for use with MinGW-based Go builds.${NC}"