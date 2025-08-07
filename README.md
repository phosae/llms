# LLM Transformers 🔄

A powerful WebAssembly-based tool for transforming API requests and responses between different LLM providers (OpenAI, Gemini, Claude).

## ✨ Features

- **🔄 N×N Transformations**: Convert between any supported providers
  - OpenAI ↔ Gemini
  - OpenAI ↔ Claude  
  - Gemini ↔ Claude
- **🌐 WebAssembly Powered**: Runs entirely in the browser, no server required
- **🎯 Interactive Web Interface**: Visual transformation with real-time validation
- **📝 Request & Response Support**: Transform both API requests and responses
- **🛠 Extensible Architecture**: Easy to add new providers
- **⚡ High Performance**: Optimized for speed and efficiency
- **📱 Responsive Design**: Works on desktop and mobile devices

## 🚀 Quick Start

### Build and Run

```bash
# Clone the repository
git clone https://github.com/phosae/llms
cd llms

# Build the WebAssembly module
make build-wasm

# Start the development server
make serve
# Or manually: cd web && python3 -m http.server 8080
```

Open your browser and navigate to `http://localhost:8080`

### Using the API

The system provides a unified interface for transforming between providers:

```go
// Create a transformation registry
registry := transformer.NewTransformationRegistry()
registry.Register(transformer.NewOpenAITransformer())
registry.Register(transformer.NewGeminiTransformer())
registry.Register(transformer.NewClaudeTransformer())

// Transform OpenAI request to Claude format
openaiRequest := &openai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []openai.ChatCompletionMessage{
        {Role: "user", Content: "Hello!"},
    },
}

claudeRequest, err := registry.Transform(
    ctx, 
    transformer.ProviderOpenAI, 
    transformer.ProviderClaude, 
    openaiRequest,
)
```

## 🏗 Architecture

### Core Components

1. **Unified Interface** (`transformer/interfaces.go`)
   - Defines common data structures for all providers
   - Provides transformation registry and management

2. **Provider Transformers**
   - `transformer/openai.go` - OpenAI API transformations
   - `transformer/gemini.go` - Google Gemini API transformations
   - `transformer/claude.go` - Anthropic Claude API transformations

3. **WebAssembly Module** (`wasm/main.go`)
   - Exposes transformation functions to JavaScript
   - Handles provider validation and examples

4. **Web Interface** (`web/`)
   - Interactive UI for transformations
   - Real-time validation and syntax highlighting
   - Example loading and output management

### Supported Features

| Feature | OpenAI | Gemini | Claude |
|---------|--------|---------|--------|
| Text Messages | ✅ | ✅ | ✅ |
| System Prompts | ✅ | ✅ | ✅ |
| Function/Tool Calls | ✅ | ✅ | ✅ |
| Image Support | ✅ | ✅ | ✅ |
| Streaming | ✅ | ✅ | ✅ |
| Temperature Control | ✅ | ✅ | ✅ |
| Max Tokens | ✅ | ✅ | ✅ |
| Stop Sequences | ✅ | ✅ | ✅ |

## 📖 API Documentation

### Unified Request Format

```go
type UnifiedRequest struct {
    Model            string                 `json:"model"`
    Messages         []UnifiedMessage       `json:"messages"`
    SystemPrompt     string                 `json:"system_prompt,omitempty"`
    MaxTokens        int                    `json:"max_tokens,omitempty"`
    Temperature      *float64               `json:"temperature,omitempty"`
    TopP             *float64               `json:"top_p,omitempty"`
    Stream           bool                   `json:"stream,omitempty"`
    Tools            []UnifiedTool          `json:"tools,omitempty"`
    ToolChoice       string                 `json:"tool_choice,omitempty"`
    StopSequences    []string               `json:"stop_sequences,omitempty"`
    // ... additional fields
}
```

### JavaScript API (WebAssembly)

```javascript
// Transform request between providers
const result = transformRequest(
    'openai',           // source provider
    'claude',           // target provider
    JSON.stringify(request)  // JSON string
);

// Get supported providers
const providers = getSupportedProviders();

// Validate request format
const validation = validateRequest('openai', requestJson);

// Get example request
const example = getExampleRequest('gemini');
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
go test -v -cover ./transformer/...

# Benchmark transformations
go test -bench=. ./transformer/...
```

## 🔧 Development

### Project Structure

```
llms/
├── dto/                    # Data Transfer Objects
│   ├── openai/            # OpenAI API structures  
│   ├── gemini/            # Gemini API structures
│   └── claude/            # Claude API structures
├── transformer/           # Core transformation logic
│   ├── interfaces.go      # Unified interfaces
│   ├── openai.go         # OpenAI transformer
│   ├── gemini.go         # Gemini transformer
│   └── claude.go         # Claude transformer
├── wasm/                  # WebAssembly entry point
│   └── main.go           
├── web/                   # Web interface
│   ├── index.html        # Main UI
│   ├── styles.css        # Styling
│   └── app.js            # JavaScript logic
└── Makefile              # Build automation
```

### Adding New Providers

1. **Define DTOs**: Create provider-specific structures in `dto/`
2. **Implement Transformer**: Create a new transformer implementing the `Transformer` interface
3. **Register Provider**: Add to registry and update constants
4. **Add Tests**: Include comprehensive test coverage
5. **Update UI**: Add provider to web interface

### WebAssembly Development

```bash
# Build WASM module
GOOS=js GOARCH=wasm go build -o web/llm-transformers.wasm ./wasm/main.go

# Copy Go WASM support files
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" web/
```

## 📊 Performance

Benchmark results on typical requests:

- **OpenAI → Claude**: ~0.5ms average
- **Gemini → OpenAI**: ~0.7ms average
- **Round-trip transformations**: ~1.2ms average
- **WASM module size**: ~8MB (gzipped: ~2MB)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run tests (`make test`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [OpenAI API Documentation](https://platform.openai.com/docs)
- [Google Gemini API Documentation](https://ai.google.dev/docs)
- [Anthropic Claude API Documentation](https://docs.anthropic.com)
- [WebAssembly Go Support](https://github.com/golang/go/wiki/WebAssembly)

## 📞 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/phosae/llms/issues)
- 💡 **Feature Requests**: [GitHub Issues](https://github.com/phosae/llms/issues)
- 📖 **Documentation**: [Wiki](https://github.com/phosae/llms/wiki)

---

**Built with ❤️ using Go and WebAssembly**
