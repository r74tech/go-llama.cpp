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
export CFLAGS="-O3 -DNDEBUG -std=c11 -fPIC -D_GLIBCXX_USE_CXX11_ABI=0"
export CXXFLAGS="-O3 -DNDEBUG -std=c++11 -fPIC -D_GLIBCXX_USE_CXX11_ABI=0"
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
    
    # List symbols to verify codecvt symbols are included
    echo -e "${GREEN}Checking for codecvt symbols:${NC}"
    if [ $IS_CROSS_COMPILE -eq 1 ]; then
        x86_64-w64-mingw32-nm libbinding.a | grep -i codecvt || echo "No codecvt symbols found (this might be normal if statically linked)"
    else
        nm libbinding.a | grep -i codecvt || echo "No codecvt symbols found (this might be normal if statically linked)"
    fi
    
    echo -e "${GREEN}Library size:${NC}"
    ls -lh libbinding.a
else
    echo -e "${RED}Build failed! libbinding.a not found.${NC}"
    exit 1
fi

echo -e "${GREEN}Done! The library is ready for use with MinGW-based Go builds.${NC}"