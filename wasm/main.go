//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/phosae/llms/dto"
	"github.com/phosae/llms/dto/gemini"
	"github.com/phosae/llms/dto/openai"
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
	var request interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		req := &openai.ChatCompletionRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse OpenAI request: %v", err))
		}
		request = req

	case transformer.ProviderGemini:
		req := &gemini.GeminiChatRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Gemini request: %v", err))
		}
		request = req

	case transformer.ProviderClaude:
		req := &dto.ClaudeRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return createErrorResult(fmt.Sprintf("Failed to parse Claude request: %v", err))
		}
		request = req

	default:
		return map[string]interface{}{
			"error": fmt.Sprintf("Unsupported source provider: %s", sourceProvider),
		}
	}

	// Create transformers directly without registry
	sourceTransformer := getTransformerForProvider(sourceProvider)
	targetTransformer := getTransformerForProvider(targetProvider)

	if sourceTransformer == nil || targetTransformer == nil {
		return createErrorResult(fmt.Sprintf("Unsupported transformation: %s -> %s", sourceProvider, targetProvider))
	}

	// Convert to unified format first
	unified, err := sourceTransformer.ToUnified(ctx, request)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to convert to unified format: %v", err))
	}

	// Transform from unified to target format
	result, err := targetTransformer.FromUnified(ctx, unified)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to convert from unified format: %v", err))
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to serialize result: %v", err),
		}
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
		return map[string]interface{}{
			"error": "Expected 3 arguments: sourceProvider, targetProvider, responseJson",
		}
	}

	sourceProvider := transformer.Provider(args[0].String())
	targetProvider := transformer.Provider(args[1].String())
	responseJsonStr := args[2].String()

	ctx := context.Background()

	// Parse the response JSON based on source provider
	var response interface{}
	var err error

	switch sourceProvider {
	case transformer.ProviderOpenAI:
		resp := &openai.ChatCompletionResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to parse OpenAI response: %v", err),
			}
		}
		response = resp

	case transformer.ProviderGemini:
		resp := &gemini.GeminiChatResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to parse Gemini response: %v", err),
			}
		}
		response = resp

	case transformer.ProviderClaude:
		resp := &dto.ClaudeResponse{}
		if err = json.Unmarshal([]byte(responseJsonStr), resp); err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to parse Claude response: %v", err),
			}
		}
		response = resp

	default:
		return map[string]interface{}{
			"error": fmt.Sprintf("Unsupported source provider: %s", sourceProvider),
		}
	}

	// Create transformers directly without registry
	sourceTransformer := getTransformerForProvider(sourceProvider)
	targetTransformer := getTransformerForProvider(targetProvider)

	if sourceTransformer == nil || targetTransformer == nil {
		return createErrorResult(fmt.Sprintf("Unsupported transformation: %s -> %s", sourceProvider, targetProvider))
	}

	// Convert response to unified format first
	unified, err := sourceTransformer.ResponseToUnified(ctx, response)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to convert response to unified format: %v", err),
		}
	}

	// Transform from unified to target format
	result, err := targetTransformer.ResponseFromUnified(ctx, unified)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to convert response from unified format: %v", err),
		}
	}

	// Convert result to JSON
	resultJson, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to serialize result: %v", err),
		}
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

	// Get transformer for the provider
	transformerInstance := getTransformerForProvider(provider)
	if transformerInstance == nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Unsupported provider: %s", provider),
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

	case transformer.ProviderGemini:
		req := &gemini.GeminiChatRequest{}
		if err = json.Unmarshal([]byte(requestJsonStr), req); err != nil {
			return map[string]interface{}{
				"error":   fmt.Sprintf("Failed to parse request: %v", err),
				"isValid": false,
			}
		}
		request = req

	case transformer.ProviderClaude:
		req := &dto.ClaudeRequest{}
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

// getTransformerForProvider returns the transformer instance for a given provider
func getTransformerForProvider(provider transformer.Provider) transformer.Transformer {
	switch provider {
	case transformer.ProviderOpenAI:
		return transformer.NewOpenAITransformer()
	case transformer.ProviderGemini:
		return transformer.NewGeminiTransformer()
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
		example = &dto.ClaudeRequest{
			Model:       "claude-3-5-sonnet-20241022",
			MaxTokens:   150,
			Temperature: &[]float64{0.7}[0],
			System:      "You are a helpful assistant.",
			Messages: []dto.ClaudeMessage{
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
