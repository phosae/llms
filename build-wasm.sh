#!/bin/bash

set -e

echo "Building LLM Transformers WASM module..."

# Set WASM build environment
export GOOS=js
export GOARCH=wasm

# Build the WASM binary
go build -o web/llm-transformers.wasm ./wasm/main.go

echo "WASM module built successfully at: web/llm-transformers.wasm"

# Copy the WASM support file from Go installation
WASM_EXEC_PATH="$(find "$(go env GOROOT)" -name "wasm_exec.js" 2>/dev/null | head -n1)"

if [ -f "$WASM_EXEC_PATH" ]; then
    cp "$WASM_EXEC_PATH" web/
    echo "Copied wasm_exec.js to web/ directory"
else
    echo "Warning: wasm_exec.js not found. Please copy it manually from your Go installation."
fi

echo "Build complete! You can now serve the web directory."