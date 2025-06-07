#!/bin/bash
set -e

# Get llama.cpp commit from release or use default
LLAMA_COMMIT=$(cat LLAMA_COMMIT 2>/dev/null || echo "ac43576124a75c2de6e333ac31a3444ff9eb9458")

echo "Setting up llama.cpp submodule at commit $LLAMA_COMMIT..."

# Clone llama.cpp with depth=1 at specific commit
if [ ! -d "llama.cpp" ]; then
    git clone --depth 1 https://github.com/ggerganov/llama.cpp.git llama.cpp
    cd llama.cpp
    git fetch --depth 1 origin $LLAMA_COMMIT
    git checkout $LLAMA_COMMIT
    cd ..
else
    echo "llama.cpp directory already exists"
    echo "Checking if it's at the correct commit..."
    cd llama.cpp
    CURRENT_COMMIT=$(git rev-parse HEAD)
    if [ "$CURRENT_COMMIT" != "$LLAMA_COMMIT" ]; then
        echo "Updating to correct commit..."
        git fetch --depth 1 origin $LLAMA_COMMIT
        git checkout $LLAMA_COMMIT
    else
        echo "Already at correct commit"
    fi
    cd ..
fi

echo "Setup complete! You can now build with Go"