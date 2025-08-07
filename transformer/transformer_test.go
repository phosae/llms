package transformer

import (
	"context"
	"testing"

	"github.com/phosae/llms/dto"
	"github.com/phosae/llms/dto/openai"
	"github.com/phosae/llms/dto/gemini"
)

func TestOpenAIToUnified(t *testing.T) {
	transformer := NewOpenAITransformer()
	
	request := &openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}
	
	ctx := context.Background()
	unified, err := transformer.ToUnified(ctx, request)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if unified.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", unified.Model)
	}
	
	if len(unified.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(unified.Messages))
	}
	
	if unified.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", unified.Messages[0].Content)
	}
}

func TestUnifiedToOpenAI(t *testing.T) {
	transformer := NewOpenAITransformer()
	
	unified := &UnifiedRequest{
		Model:     "gpt-4",
		MaxTokens: 100,
		Messages: []UnifiedMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
	}
	
	ctx := context.Background()
	result, err := transformer.FromUnified(ctx, unified)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	request, ok := result.(*openai.ChatCompletionRequest)
	if !ok {
		t.Fatalf("Expected *openai.ChatCompletionRequest, got %T", result)
	}
	
	if request.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", request.Model)
	}
	
	if len(request.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(request.Messages))
	}
	
	if request.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", request.Messages[0].Content)
	}
}

func TestGeminiToUnified(t *testing.T) {
	transformer := NewGeminiTransformer()
	
	request := &gemini.GeminiChatRequest{
		Contents: []gemini.GeminiChatContent{
			{
				Role: "user",
				Parts: []gemini.GeminiPart{
					{Text: "Hello, world!"},
				},
			},
		},
	}
	
	ctx := context.Background()
	unified, err := transformer.ToUnified(ctx, request)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(unified.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(unified.Messages))
	}
	
	if unified.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", unified.Messages[0].Content)
	}
}

func TestClaudeToUnified(t *testing.T) {
	transformer := NewClaudeTransformer()
	
	request := &dto.ClaudeRequest{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 100,
		Messages: []dto.ClaudeMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
	}
	
	ctx := context.Background()
	unified, err := transformer.ToUnified(ctx, request)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if unified.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", unified.Model)
	}
	
	if len(unified.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(unified.Messages))
	}
	
	if unified.Messages[0].Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", unified.Messages[0].Content)
	}
}

func TestTransformationRegistry(t *testing.T) {
	registry := NewTransformationRegistry()
	
	// Register transformers
	registry.Register(NewOpenAITransformer())
	registry.Register(NewGeminiTransformer())
	registry.Register(NewClaudeTransformer())
	
	// Test getting supported providers
	providers := registry.GetSupportedProviders()
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}
	
	// Test transformation
	request := &openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		MaxTokens: 100,
	}
	
	ctx := context.Background()
	result, err := registry.Transform(ctx, ProviderOpenAI, ProviderClaude, request)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	claudeRequest, ok := result.(*dto.ClaudeRequest)
	if !ok {
		t.Fatalf("Expected *dto.ClaudeRequest, got %T", result)
	}
	
	if len(claudeRequest.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(claudeRequest.Messages))
	}
}

func TestRoundTripTransformation(t *testing.T) {
	registry := NewTransformationRegistry()
	registry.Register(NewOpenAITransformer())
	registry.Register(NewGeminiTransformer())
	registry.Register(NewClaudeTransformer())
	
	// Original OpenAI request
	original := &openai.ChatCompletionRequest{
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
	}
	
	ctx := context.Background()
	
	// Transform OpenAI -> Gemini -> OpenAI
	geminiResult, err := registry.Transform(ctx, ProviderOpenAI, ProviderGemini, original)
	if err != nil {
		t.Fatalf("OpenAI -> Gemini transformation failed: %v", err)
	}
	
	openaiResult, err := registry.Transform(ctx, ProviderGemini, ProviderOpenAI, geminiResult)
	if err != nil {
		t.Fatalf("Gemini -> OpenAI transformation failed: %v", err)
	}
	
	finalRequest, ok := openaiResult.(*openai.ChatCompletionRequest)
	if !ok {
		t.Fatalf("Expected *openai.ChatCompletionRequest, got %T", openaiResult)
	}
	
	// Verify the essential data is preserved (model may change during transformation)
	if len(finalRequest.Messages) != len(original.Messages) {
		t.Errorf("Message count not preserved: expected %d, got %d", len(original.Messages), len(finalRequest.Messages))
	}
}

func BenchmarkTransformation(b *testing.B) {
	registry := NewTransformationRegistry()
	registry.Register(NewOpenAITransformer())
	registry.Register(NewGeminiTransformer())
	registry.Register(NewClaudeTransformer())
	
	request := &openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		MaxTokens: 100,
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Transform(ctx, ProviderOpenAI, ProviderClaude, request)
		if err != nil {
			b.Fatalf("Transformation failed: %v", err)
		}
	}
}