//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/phosae/llms/claude"
	"github.com/phosae/llms/gemini"
	"github.com/phosae/llms/openai"
	"github.com/phosae/llms/transformer"
)

// supportedProviders defines the list once to avoid duplication
var supportedProviders = []string{"openai", "gemini", "claude"}

// createErrorResult is a helper to create consistent error responses
func createErrorResult(message string) map[string]interface{} {
	return map[string]interface{}{
		"error": message,
	}
}

// transformRequest transforms a request from source provider to target provider
func transformRequest(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in transformRequest: %v\n", r)
		}
	}()

	if len(args) != 3 {
		return createErrorResult("Expected 3 arguments: sourceProvider, targetProvider, requestJson")
	}

	sourceProvider := transformer.Provider(args[0].String())
	targetProvider := transformer.Provider(args[1].String())
	requestJsonStr := args[2].String()

	fmt.Printf("Transform request: %s -> %s\n", sourceProvider, targetProvider)
	ctx := context.Background()

	// Parse the request JSON based on source provider
	var srcRequest interface{}
	var dstRequest interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		req := &openai.ChatCompletionRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse OpenAI request: %v", err))
		}
		srcRequest = req

	case transformer.ProviderGemini:
		req := &gemini.GeminiChatRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Gemini request: %v", err))
		}
		srcRequest = req

	case transformer.ProviderClaude:
		req := &claude.ClaudeRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Claude request: %v", err))
		}
		srcRequest = req

	default:
		return createErrorResult(fmt.Sprintf("Unsupported source provider: %s", sourceProvider))
	}

	// Create destination request object
	switch targetProvider {
	case transformer.ProviderOpenAI:
		dstRequest = &openai.ChatCompletionRequest{}
	case transformer.ProviderGemini:
		dstRequest = &gemini.GeminiChatRequest{}
	case transformer.ProviderClaude:
		dstRequest = &claude.ClaudeRequest{}
	default:
		return createErrorResult(fmt.Sprintf("Unsupported target provider: %s", targetProvider))
	}

	// Get direct transformer
	transformerInstance := getDirectTransformer(sourceProvider)
	if transformerInstance == nil {
		return createErrorResult(fmt.Sprintf("No transformer available for %s -> %s", sourceProvider, targetProvider))
	}

	// Perform direct transformation
	err = transformerInstance.Do(ctx, transformer.TransformerTypeRequest, srcRequest, dstRequest)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to transform request: %v", err))
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(dstRequest, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to serialize result: %v", err))
	}

	return map[string]interface{}{
		"success": true,
		"result":  string(resultJson),
	}
}

// transformResponse transforms a response from source provider to target provider
func transformResponse(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in transformResponse: %v\n", r)
		}
	}()

	if len(args) != 3 {
		return createErrorResult("Expected 3 arguments: sourceProvider, targetProvider, responseJson")
	}

	sourceProvider := transformer.Provider(args[0].String())
	targetProvider := transformer.Provider(args[1].String())
	responseJsonStr := args[2].String()

	ctx := context.Background()

	// Parse the response JSON based on source provider
	var srcResponse interface{}
	var dstResponse interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		resp := &openai.ChatCompletionResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse OpenAI response: %v", err))
		}
		srcResponse = resp

	case transformer.ProviderGemini:
		resp := &gemini.GeminiChatResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Gemini response: %v", err))
		}
		srcResponse = resp

	case transformer.ProviderClaude:
		resp := &claude.ClaudeResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Claude response: %v", err))
		}
		srcResponse = resp

	default:
		return createErrorResult(fmt.Sprintf("Unsupported source provider: %s", sourceProvider))
	}

	// Create destination response object
	switch targetProvider {
	case transformer.ProviderOpenAI:
		dstResponse = &openai.ChatCompletionResponse{}
	case transformer.ProviderGemini:
		dstResponse = &gemini.GeminiChatResponse{}
	case transformer.ProviderClaude:
		dstResponse = &claude.ClaudeResponse{}
	default:
		return createErrorResult(fmt.Sprintf("Unsupported target provider: %s", targetProvider))
	}

	// Get direct transformer
	transformerInstance := getDirectTransformer(sourceProvider)
	if transformerInstance == nil {
		return createErrorResult(fmt.Sprintf("No transformer available for %s -> %s", sourceProvider, targetProvider))
	}

	// Perform direct transformation
	err = transformerInstance.Do(ctx, transformer.TransformerTypeResponse, srcResponse, dstResponse)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to transform response: %v", err))
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(dstResponse, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to serialize result: %v", err))
	}

	return map[string]interface{}{
		"success": true,
		"result":  string(resultJson),
	}
}

// transformStream transforms a full stream response from source provider to target provider
func transformStream(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in transformStream: %v\n", r)
		}
	}()

	if len(args) != 3 {
		return createErrorResult("Expected 3 arguments: sourceProvider, targetProvider, streamJson")
	}

	sourceProvider := transformer.Provider(args[0].String())
	targetProvider := transformer.Provider(args[1].String())
	streamJsonStr := args[2].String()

	ctx := context.Background()

	// Parse the stream JSON based on source provider
	var srcStream interface{}
	var dstStream interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		stream := &openai.ChatCompletionStreamResponse{}
		if err = json.Unmarshal([]byte(streamJsonStr), stream); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse OpenAI stream: %v", err))
		}
		srcStream = stream

	case transformer.ProviderClaude:
		stream := &claude.ClaudeResponse{}
		if err = json.Unmarshal([]byte(streamJsonStr), stream); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Claude stream: %v", err))
		}
		srcStream = stream

	default:
		return createErrorResult(fmt.Sprintf("Unsupported source provider for stream: %s", sourceProvider))
	}

	// Create destination stream object
	switch targetProvider {
	case transformer.ProviderOpenAI:
		dstStream = &openai.ChatCompletionStreamResponse{}
	case transformer.ProviderClaude:
		dstStream = &claude.ClaudeResponse{}
	default:
		return createErrorResult(fmt.Sprintf("Unsupported target provider for stream: %s", targetProvider))
	}

	// Get direct transformer
	transformerInstance := getDirectTransformer(sourceProvider)
	if transformerInstance == nil {
		return createErrorResult(fmt.Sprintf("No transformer available for %s -> %s stream", sourceProvider, targetProvider))
	}

	// Perform direct stream transformation
	err = transformerInstance.Do(ctx, transformer.TransformerTypeStream, srcStream, dstStream)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to transform stream: %v", err))
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(dstStream, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to serialize stream result: %v", err))
	}

	return map[string]interface{}{
		"success": true,
		"result":  string(resultJson),
	}
}

// transformChunk transforms a single chunk from stream response
func transformChunk(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in transformChunk: %v\n", r)
		}
	}()

	if len(args) != 3 {
		return createErrorResult("Expected 3 arguments: sourceProvider, targetProvider, chunkJson")
	}

	sourceProvider := transformer.Provider(args[0].String())
	targetProvider := transformer.Provider(args[1].String())
	chunkJsonStr := args[2].String()

	ctx := context.Background()

	// Parse the chunk JSON based on source provider
	var srcChunk interface{}
	var dstChunk interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		chunk := &openai.ChatCompletionStreamResponse{}
		if err = json.Unmarshal([]byte(chunkJsonStr), chunk); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse OpenAI chunk: %v", err))
		}
		srcChunk = chunk

	case transformer.ProviderClaude:
		chunk := &claude.ClaudeResponse{}
		if err = json.Unmarshal([]byte(chunkJsonStr), chunk); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Claude chunk: %v", err))
		}
		srcChunk = chunk

	default:
		return createErrorResult(fmt.Sprintf("Unsupported source provider for chunk: %s", sourceProvider))
	}

	// Create destination chunk object
	switch targetProvider {
	case transformer.ProviderOpenAI:
		dstChunk = &openai.ChatCompletionStreamResponse{}
	case transformer.ProviderClaude:
		dstChunk = &claude.ClaudeResponse{}
	default:
		return createErrorResult(fmt.Sprintf("Unsupported target provider for chunk: %s", targetProvider))
	}

	// Get direct transformer
	transformerInstance := getDirectTransformer(sourceProvider)
	if transformerInstance == nil {
		return createErrorResult(fmt.Sprintf("No transformer available for %s -> %s chunk", sourceProvider, targetProvider))
	}

	// Perform direct chunk transformation
	err = transformerInstance.Do(ctx, transformer.TransformerTypeChunk, srcChunk, dstChunk)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to transform chunk: %v", err))
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(dstChunk, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to serialize chunk result: %v", err))
	}

	return map[string]interface{}{
		"success": true,
		"result":  string(resultJson),
	}
}

// getSupportedProviders returns all supported providers
func getSupportedProviders(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in getSupportedProviders: %v\n", r)
		}
	}()

	fmt.Println("getSupportedProviders called")

	return map[string]interface{}{
		"success":   true,
		"providers": supportedProviders,
	}
}

// getAvailableTransformations returns all available transformation pairs
func getAvailableTransformations(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in getAvailableTransformations: %v\n", r)
		}
	}()

	fmt.Println("getAvailableTransformations called")

	// Create all possible transformation pairs manually to avoid registry issues
	var transformationPairs []map[string]interface{}

	for _, source := range supportedProviders {
		for _, target := range supportedProviders {
			if source != target {
				transformationPairs = append(transformationPairs, map[string]interface{}{
					"source": source,
					"target": target,
				})
			}
		}
	}

	// Convert to JSON string first to ensure compatibility
	result := map[string]interface{}{
		"success":         true,
		"transformations": transformationPairs,
	}

	resultJson, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("Failed to marshal transformations: %v\n", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to serialize transformations: %v", err),
		}
	}

	// Return as JSON string to avoid syscall/js.ValueOf issues
	var parsedResult map[string]interface{}
	if err := json.Unmarshal(resultJson, &parsedResult); err != nil {
		fmt.Printf("Failed to unmarshal transformations: %v\n", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse transformations: %v", err),
		}
	}

	return parsedResult
}

// validateRequest validates a request for a specific provider
func validateRequest(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{
			"error": "Expected 2 arguments: provider, requestJson",
		}
	}

	provider := transformer.Provider(args[0].String())
	requestJsonStr := args[1].String()
	ctx := context.Background()

	// Get any transformer that can validate this provider's requests
	var transformerInstance transformer.Transformer
	switch provider {
	case transformer.ProviderOpenAI:
		transformerInstance = transformer.NewOpenAITransformer()
	case transformer.ProviderClaude:
		transformerInstance = transformer.NewClaudeTransformer()
	default:
		return map[string]interface{}{
			"error":   fmt.Sprintf("Unsupported provider: %s", provider),
			"isValid": false,
		}
	}

	// Parse the request JSON
	var request interface{}
	var err error

	switch provider {
	case transformer.ProviderOpenAI:
		req := &openai.ChatCompletionRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return map[string]interface{}{
				"error":   fmt.Sprintf("Failed to parse request: %v", err),
				"isValid": false,
			}
		}
		request = req

	case transformer.ProviderClaude:
		req := &claude.ClaudeRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return map[string]interface{}{
				"error":   fmt.Sprintf("Failed to parse request: %v", err),
				"isValid": false,
			}
		}
		request = req
	}

	// Validate the request
	if err := transformerInstance.ValidateRequest(ctx, request); err != nil {
		return map[string]interface{}{
			"error":   fmt.Sprintf("Validation failed: %v", err),
			"isValid": false,
		}
	}

	return map[string]interface{}{
		"success": true,
		"isValid": true,
	}
}

// getDirectTransformer returns the direct transformer for a specific source->target pair
func getDirectTransformer(sourceProvider transformer.Provider) transformer.Transformer {
	switch sourceProvider {
	case transformer.ProviderOpenAI:
		return transformer.NewOpenAITransformer()
	case transformer.ProviderClaude:
		return transformer.NewClaudeTransformer()
	default:
		return nil
	}
}

// getExampleRequest returns an example request for a provider
func getExampleRequest(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{
			"error": "Expected 1 argument: provider",
		}
	}

	provider := transformer.Provider(args[0].String())

	var example interface{}

	switch provider {
	case transformer.ProviderOpenAI:
		example = &openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: "Hello, how are you?",
				},
			},
			MaxTokens:   150,
			Temperature: 0.7,
			TopP:        1.0,
		}

	case transformer.ProviderGemini:
		example = &gemini.GeminiChatRequest{
			Contents: []gemini.GeminiChatContent{
				{
					Role: "user",
					Parts: []gemini.GeminiPart{
						{Text: "Hello, how are you?"},
					},
				},
			},
			SystemInstructions: &gemini.GeminiChatContent{
				Parts: []gemini.GeminiPart{
					{Text: "You are a helpful assistant."},
				},
			},
			GenerationConfig: gemini.GeminiChatGenerationConfig{
				MaxOutputTokens: 150,
				Temperature:     &[]float64{0.7}[0],
				TopP:            1.0,
			},
		}

	case transformer.ProviderClaude:
		example = &claude.ClaudeRequest{
			Model:       "claude-3-5-sonnet-20241022",
			MaxTokens:   150,
			Temperature: &[]float64{0.7}[0],
			System:      "You are a helpful assistant.",
			Messages: []claude.ClaudeMessage{
				{
					Role:    "user",
					Content: "Hello, how are you?",
				},
			},
		}

	default:
		return map[string]interface{}{
			"error": fmt.Sprintf("Unsupported provider: %s", provider),
		}
	}

	// Convert to JSON
	exampleJson, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to serialize example: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"example": string(exampleJson),
	}
}

func main() {
	// Add panic recovery for the main function
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in main: %v\n", r)
			// Don't try to continue after panic in main
		}
	}()

	fmt.Println("Starting LLM Transformer WASM module...")

	// Safely register JavaScript functions with error handling
	safeRegister := func(name string, fn func(js.Value, []js.Value) interface{}) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Failed to register function %s: %v\n", name, r)
			}
		}()
		js.Global().Set(name, js.FuncOf(fn))
		fmt.Printf("Registered function: %s\n", name)
	}

	safeRegister("transformRequest", transformRequest)
	safeRegister("transformResponse", transformResponse)
	safeRegister("transformStream", transformStream)
	safeRegister("transformChunk", transformChunk)
	safeRegister("getSupportedProviders", getSupportedProviders)
	safeRegister("getAvailableTransformations", getAvailableTransformations)
	safeRegister("validateRequest", validateRequest)
	safeRegister("getExampleRequest", getExampleRequest)

	fmt.Println("All JavaScript functions registered successfully")

	// Signal that WASM module is ready
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Failed to send ready message: %v\n", r)
		}
	}()

	js.Global().Get("window").Call("postMessage", map[string]interface{}{
		"type":    "wasmReady",
		"message": "LLM transformer WASM module loaded successfully",
	}, "*")

	fmt.Println("WASM ready message sent")

	// Keep the main function running indefinitely
	fmt.Println("WASM module ready and waiting for function calls...")

	// Use a blocking channel instead of setTimeout to avoid runtime issues
	done := make(chan struct{})
	<-done
}
