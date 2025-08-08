package transformer

import (
	"context"
	"fmt"

	"github.com/phosae/llms/gemini"
)

// GeminiTransformer handles Gemini format transformations
type GeminiTransformer struct{}

// NewGeminiTransformer creates a new Gemini transformer
func NewGeminiTransformer() *GeminiTransformer {
	return &GeminiTransformer{}
}

// GetProvider returns the provider this transformer handles
func (t *GeminiTransformer) GetProvider() Provider {
	return ProviderGemini
}

// ValidateRequest validates the Gemini request
func (t *GeminiTransformer) ValidateRequest(ctx context.Context, request interface{}) error {
	req, ok := request.(*gemini.GeminiChatRequest)
	if !ok {
		return fmt.Errorf("invalid request type for Gemini transformer")
	}

	if len(req.Contents) == 0 {
		return fmt.Errorf("contents cannot be empty")
	}

	return nil
}

// ToUnified converts Gemini request to unified format
func (t *GeminiTransformer) ToUnified(ctx context.Context, providerRequest interface{}) (*UnifiedRequest, error) {
	req, ok := providerRequest.(*gemini.GeminiChatRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Gemini transformer")
	}

	if err := t.ValidateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	unified := &UnifiedRequest{
		Model: "gemini", // Default model name, should be set externally if different
	}

	// Handle generation config
	if req.GenerationConfig.Temperature != nil {
		unified.Temperature = req.GenerationConfig.Temperature
	}
	if req.GenerationConfig.TopP > 0 {
		topP := req.GenerationConfig.TopP
		unified.TopP = &topP
	}
	if req.GenerationConfig.MaxOutputTokens > 0 {
		unified.MaxTokens = int(req.GenerationConfig.MaxOutputTokens)
	}
	if len(req.GenerationConfig.StopSequences) > 0 {
		unified.StopSequences = req.GenerationConfig.StopSequences
	}
	if req.GenerationConfig.Seed != 0 {
		unified.Seed = &req.GenerationConfig.Seed
	}

	// Handle system instructions
	if req.SystemInstructions != nil {
		unified.SystemPrompt = t.extractTextFromParts(req.SystemInstructions.Parts)
	}

	// Convert contents to messages
	for _, content := range req.Contents {
		unifiedMsg := UnifiedMessage{
			Role: t.convertGeminiRoleToUnified(content.Role),
		}

		// Handle parts
		if len(content.Parts) > 0 {
			textContent := t.extractTextFromParts(content.Parts)
			if textContent != "" {
				unifiedMsg.Content = textContent
			}

			// Convert parts with non-text content
			for _, part := range content.Parts {
				if part.Text == "" { // Non-text parts
					unifiedPart := t.convertGeminiPartToUnified(part)
					if unifiedPart != nil {
						unifiedMsg.Parts = append(unifiedMsg.Parts, *unifiedPart)
					}
				}
			}

			// Handle function calls
			for _, part := range content.Parts {
				if part.FunctionCall != nil {
					args := convertAnyToMap(part.FunctionCall.Arguments)
					unifiedMsg.ToolCalls = append(unifiedMsg.ToolCalls, UnifiedToolCall{
						ID:        generateToolCallID(),
						Type:      "function",
						Name:      part.FunctionCall.FunctionName,
						Arguments: args,
					})
				}
			}
		}

		unified.Messages = append(unified.Messages, unifiedMsg)
	}

	// Convert tools
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			// Handle built-in tools
			if tool.CodeExecution != nil {
				unifiedTool := UnifiedTool{
					Type:        "code_execution",
					Name:        "codeExecution",
					Description: "Execute code in a secure environment",
				}
				unified.Tools = append(unified.Tools, unifiedTool)
			}

			if tool.GoogleSearch != nil {
				unifiedTool := UnifiedTool{
					Type:        "google_search",
					Name:        "googleSearch",
					Description: "Search the web using Google Search",
				}
				unified.Tools = append(unified.Tools, unifiedTool)
			}

			if tool.GoogleSearchRetrieval != nil {
				unifiedTool := UnifiedTool{
					Type:        "google_search_retrieval",
					Name:        "googleSearchRetrieval",
					Description: "Search and retrieve information using Google Search with enhanced retrieval capabilities",
				}
				unified.Tools = append(unified.Tools, unifiedTool)
			}

			// Handle function declarations
			if tool.FunctionDeclarations != nil {
				if funcs, ok := tool.FunctionDeclarations.([]interface{}); ok {
					for _, f := range funcs {
						if funcMap, ok := f.(map[string]interface{}); ok {
							unifiedTool := UnifiedTool{
								Type:        "function",
								Name:        getStringFromMap(funcMap, "name"),
								Description: getStringFromMap(funcMap, "description"),
								Parameters:  convertAnyToMap(funcMap["parameters"]),
							}
							unified.Tools = append(unified.Tools, unifiedTool)
						}
					}
				}
			}
		}
	}

	return unified, nil
}

// FromUnified converts unified request to Gemini format
func (t *GeminiTransformer) FromUnified(ctx context.Context, unifiedRequest *UnifiedRequest) (interface{}, error) {
	req := &gemini.GeminiChatRequest{}

	// Handle generation config
	req.GenerationConfig = gemini.GeminiChatGenerationConfig{}
	if unifiedRequest.Temperature != nil {
		req.GenerationConfig.Temperature = unifiedRequest.Temperature
	}
	if unifiedRequest.TopP != nil {
		req.GenerationConfig.TopP = *unifiedRequest.TopP
	}
	if unifiedRequest.MaxTokens > 0 {
		req.GenerationConfig.MaxOutputTokens = uint(unifiedRequest.MaxTokens)
	}
	if len(unifiedRequest.StopSequences) > 0 {
		req.GenerationConfig.StopSequences = unifiedRequest.StopSequences
	}
	if unifiedRequest.Seed != nil {
		req.GenerationConfig.Seed = *unifiedRequest.Seed
	}

	// Handle system prompt
	if unifiedRequest.SystemPrompt != "" {
		req.SystemInstructions = &gemini.GeminiChatContent{
			Parts: []gemini.GeminiPart{
				{Text: unifiedRequest.SystemPrompt},
			},
		}
	}

	// Convert messages to contents
	for _, unifiedMsg := range unifiedRequest.Messages {
		content := gemini.GeminiChatContent{
			Role: t.convertUnifiedRoleToGemini(unifiedMsg.Role),
		}

		// Handle text content
		if unifiedMsg.Content != "" {
			content.Parts = append(content.Parts, gemini.GeminiPart{
				Text: unifiedMsg.Content,
			})
		}

		// Handle multipart content
		for _, part := range unifiedMsg.Parts {
			geminiPart := t.convertUnifiedPartToGemini(part)
			if geminiPart != nil {
				content.Parts = append(content.Parts, *geminiPart)
			}
		}

		// Handle tool calls
		for _, toolCall := range unifiedMsg.ToolCalls {
			content.Parts = append(content.Parts, gemini.GeminiPart{
				FunctionCall: &gemini.FunctionCall{
					FunctionName: toolCall.Name,
					Arguments:    toolCall.Arguments,
				},
			})
		}

		req.Contents = append(req.Contents, content)
	}

	// Convert tools
	if len(unifiedRequest.Tools) > 0 {
		var functionDeclarations []interface{}

		for _, unifiedTool := range unifiedRequest.Tools {
			// First check if this should be a built-in tool by name, regardless of type
			switch unifiedTool.Name {
			case "code_interpreter", "python", "code_execution", "codeExecution":
				req.Tools = append(req.Tools, gemini.GeminiChatTool{
					CodeExecution: map[string]interface{}{},
				})
			case "web_search", "google_search", "googleSearch", "search":
				req.Tools = append(req.Tools, gemini.GeminiChatTool{
					GoogleSearch: map[string]interface{}{},
				})
			case "google_search_retrieval", "googleSearchRetrieval":
				req.Tools = append(req.Tools, gemini.GeminiChatTool{
					GoogleSearchRetrieval: map[string]interface{}{},
				})
			default:
				// Then check by type
				switch unifiedTool.Type {
				case "code_execution", "codeExecution":
					// Gemini built-in code execution tool
					req.Tools = append(req.Tools, gemini.GeminiChatTool{
						CodeExecution: map[string]interface{}{},
					})

				case "google_search", "googleSearch":
					// Gemini built-in Google search tool
					req.Tools = append(req.Tools, gemini.GeminiChatTool{
						GoogleSearch: map[string]interface{}{},
					})

				case "google_search_retrieval", "googleSearchRetrieval":
					// Gemini built-in Google search retrieval tool
					req.Tools = append(req.Tools, gemini.GeminiChatTool{
						GoogleSearchRetrieval: map[string]interface{}{},
					})

				default:
					// Treat as regular function declaration
					funcDecl := map[string]interface{}{
						"name":        unifiedTool.Name,
						"description": unifiedTool.Description,
						"parameters":  unifiedTool.Parameters,
					}
					functionDeclarations = append(functionDeclarations, funcDecl)
				}
			}
		}

		// Add function declarations if any exist
		if len(functionDeclarations) > 0 {
			req.Tools = append(req.Tools, gemini.GeminiChatTool{
				FunctionDeclarations: functionDeclarations,
			})
		}
	}

	return req, nil
}

// ResponseToUnified converts Gemini response to unified format
func (t *GeminiTransformer) ResponseToUnified(ctx context.Context, providerResponse interface{}) (*UnifiedResponse, error) {
	resp, ok := providerResponse.(*gemini.GeminiChatResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type for Gemini transformer")
	}

	unified := &UnifiedResponse{
		Provider: ProviderGemini,
		Object:   "chat.completion",
	}

	// Convert usage
	if resp.UsageMetadata.TotalTokenCount > 0 {
		unified.Usage = &UnifiedUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	// Convert candidates to choices
	for i, candidate := range resp.Candidates {
		unifiedChoice := UnifiedChoice{
			Index: i,
		}

		// Convert message
		unifiedChoice.Message = UnifiedMessage{
			Role: t.convertGeminiRoleToUnified(candidate.Content.Role),
		}

		// Extract text content
		textContent := t.extractTextFromParts(candidate.Content.Parts)
		if textContent != "" {
			unifiedChoice.Message.Content = textContent
		}

		// Handle non-text parts
		for _, part := range candidate.Content.Parts {
			if part.Text == "" {
				unifiedPart := t.convertGeminiPartToUnified(part)
				if unifiedPart != nil {
					unifiedChoice.Message.Parts = append(unifiedChoice.Message.Parts, *unifiedPart)
				}
			}

			// Handle function calls
			if part.FunctionCall != nil {
				args := convertAnyToMap(part.FunctionCall.Arguments)
				unifiedChoice.Message.ToolCalls = append(unifiedChoice.Message.ToolCalls, UnifiedToolCall{
					ID:        generateToolCallID(),
					Type:      "function",
					Name:      part.FunctionCall.FunctionName,
					Arguments: args,
				})
			}
		}

		// Convert finish reason
		if candidate.FinishReason != nil {
			unifiedChoice.FinishReason = t.convertGeminiFinishReason(*candidate.FinishReason)
		}

		unified.Choices = append(unified.Choices, unifiedChoice)
	}

	return unified, nil
}

// ResponseFromUnified converts unified response to Gemini format
func (t *GeminiTransformer) ResponseFromUnified(ctx context.Context, unifiedResponse *UnifiedResponse) (interface{}, error) {
	resp := &gemini.GeminiChatResponse{}

	// Convert usage
	if unifiedResponse.Usage != nil {
		resp.UsageMetadata = gemini.GeminiUsageMetadata{
			PromptTokenCount:     unifiedResponse.Usage.PromptTokens,
			CandidatesTokenCount: unifiedResponse.Usage.CompletionTokens,
			TotalTokenCount:      unifiedResponse.Usage.TotalTokens,
		}
	}

	// Convert choices to candidates
	for _, unifiedChoice := range unifiedResponse.Choices {
		candidate := gemini.GeminiChatCandidate{
			Index: int64(unifiedChoice.Index),
		}

		// Convert message
		candidate.Content = gemini.GeminiChatContent{
			Role: t.convertUnifiedRoleToGemini(unifiedChoice.Message.Role),
		}

		// Handle text content
		if unifiedChoice.Message.Content != "" {
			candidate.Content.Parts = append(candidate.Content.Parts, gemini.GeminiPart{
				Text: unifiedChoice.Message.Content,
			})
		}

		// Handle multipart content
		for _, part := range unifiedChoice.Message.Parts {
			geminiPart := t.convertUnifiedPartToGemini(part)
			if geminiPart != nil {
				candidate.Content.Parts = append(candidate.Content.Parts, *geminiPart)
			}
		}

		// Handle tool calls
		for _, toolCall := range unifiedChoice.Message.ToolCalls {
			candidate.Content.Parts = append(candidate.Content.Parts, gemini.GeminiPart{
				FunctionCall: &gemini.FunctionCall{
					FunctionName: toolCall.Name,
					Arguments:    toolCall.Arguments,
				},
			})
		}

		// Convert finish reason
		if unifiedChoice.FinishReason != "" {
			finishReason := t.convertUnifiedFinishReasonToGemini(unifiedChoice.FinishReason)
			candidate.FinishReason = &finishReason
		}

		resp.Candidates = append(resp.Candidates, candidate)
	}

	return resp, nil
}

// Helper functions

func (t *GeminiTransformer) convertGeminiRoleToUnified(role string) string {
	switch role {
	case "user":
		return "user"
	case "model":
		return "assistant"
	case "function":
		return "tool"
	default:
		return role
	}
}

func (t *GeminiTransformer) convertUnifiedRoleToGemini(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "model"
	case "system":
		return "user" // Gemini doesn't have system role, treat as user
	case "tool":
		return "function"
	default:
		return role
	}
}

func (t *GeminiTransformer) extractTextFromParts(parts []gemini.GeminiPart) string {
	var text string
	for _, part := range parts {
		if part.Text != "" {
			text += part.Text
		}
	}
	return text
}

func (t *GeminiTransformer) convertGeminiPartToUnified(part gemini.GeminiPart) *UnifiedMessagePart {
	if part.InlineData != nil {
		return &UnifiedMessagePart{
			Type:      "image",
			MediaType: part.InlineData.MimeType,
			Data:      part.InlineData.Data,
		}
	}

	if part.FileData != nil {
		return &UnifiedMessagePart{
			Type:      "file",
			MediaType: part.FileData.MimeType,
			Data:      part.FileData.FileUri,
		}
	}

	return nil
}

func (t *GeminiTransformer) convertUnifiedPartToGemini(part UnifiedMessagePart) *gemini.GeminiPart {
	switch part.Type {
	case "image":
		if part.MediaType != "" && part.Data != "" {
			return &gemini.GeminiPart{
				InlineData: &gemini.GeminiInlineData{
					MimeType: part.MediaType,
					Data:     part.Data,
				},
			}
		}
		if part.ImageURL != nil {
			// For URL-based images, we'd need to download and encode
			// For now, just return nil as this requires external HTTP calls
			return nil
		}
	case "file":
		if part.MediaType != "" && part.Data != "" {
			return &gemini.GeminiPart{
				FileData: &gemini.GeminiFileData{
					MimeType: part.MediaType,
					FileUri:  part.Data,
				},
			}
		}
	}
	return nil
}

func (t *GeminiTransformer) convertGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return reason
	}
}

func (t *GeminiTransformer) convertUnifiedFinishReasonToGemini(reason string) string {
	switch reason {
	case "stop":
		return "STOP"
	case "length":
		return "MAX_TOKENS"
	case "content_filter":
		return "SAFETY"
	default:
		return "STOP"
	}
}

// Helper functions
func generateToolCallID() string {
	// Simple ID generation - in production, use proper UUID
	return fmt.Sprintf("call_%d", len(fmt.Sprintf("%d", 1234567890)))
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
