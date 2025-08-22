package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/phosae/llms/claude"
	"github.com/phosae/llms/gemini"
	"github.com/phosae/llms/openai"
)

// OpenAITransformer handles direct OpenAI to other provider's transformations
type OpenAITransformer struct{}

// NewOpenAITransformer creates a new OpenAI to other provider's transformer
func NewOpenAITransformer() *OpenAITransformer {
	return &OpenAITransformer{}
}

// GetProvider returns the source provider (OpenAI)
func (t *OpenAITransformer) GetProvider() Provider {
	return ProviderOpenAI
}

// ValidateRequest validates the OpenAI request
func (t *OpenAITransformer) ValidateRequest(ctx context.Context, request interface{}) error {
	req, ok := request.(*openai.ChatCompletionRequest)
	if !ok {
		return fmt.Errorf("invalid request type for OpenAI transformer")
	}

	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	return nil
}

// Do performs the transformation based on the type
func (t *OpenAITransformer) Do(ctx context.Context, typ TransformerType, src interface{}, dst interface{}) error {
	switch typ {
	case TransformerTypeRequest:
		return t.transformRequest(ctx, src, dst)
	case TransformerTypeResponse:
		return t.transformResponse(ctx, src, dst)
	case TransformerTypeStream:
		return t.transformStreamResponse(ctx, src, dst)
	case TransformerTypeChunk:
		return t.transformChunk(ctx, src, dst)
	default:
		return fmt.Errorf("unsupported transformation type: %s", typ)
	}
}

func (t *OpenAITransformer) transformRequest(ctx context.Context, src interface{}, dst interface{}) error {
	oaiReq, ok := src.(*openai.ChatCompletionRequest)
	if !ok {
		return fmt.Errorf("invalid source type for OpenAI transformer")
	}

	switch target := dst.(type) {
	case *claude.ClaudeRequest:
		return transformRequestToClaude(ctx, oaiReq, target)
	case *gemini.GeminiChatRequest:
		return transformRequestToGemini(ctx, oaiReq, target)
	default:
		return fmt.Errorf("target type not supported for OpenAI transformer")
	}
}

func (t *OpenAITransformer) transformResponse(ctx context.Context, src interface{}, dst interface{}) error {
	oaiResp, ok := src.(*openai.ChatCompletionResponse)
	if !ok {
		return fmt.Errorf("invalid source type for OpenAI transformer")
	}

	switch dst.(type) {
	case *claude.ClaudeResponse:
		return transformResponseToClaude(ctx, oaiResp, dst.(*claude.ClaudeResponse))
	default:
		return fmt.Errorf("target type not supported for OpenAI transformer")
	}
}

func transformResponseToClaude(ctx context.Context, oaiResp *openai.ChatCompletionResponse, claudeResp *claude.ClaudeResponse) error {
	claudeResp.Id = oaiResp.ID
	claudeResp.Type = "message"
	claudeResp.Role = "assistant"
	claudeResp.Model = oaiResp.Model

	for _, choice := range oaiResp.Choices {
		claudeResp.StopReason = stopReasonOpenAI2Claude(string(choice.FinishReason))
		if choice.FinishReason == "tool_calls" {
			for _, toolCall := range choice.Message.ToolCalls {
				claudeResp.Content = append(claudeResp.Content, claude.ClaudeMediaMessage{
					Type:  "tool_use",
					Id:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: toolCall.Function.Arguments,
				})
			}
		} else {
			claudeResp.Content = append(claudeResp.Content, claude.ClaudeMediaMessage{
				Type: "text",
				Text: &choice.Message.Content,
			})
		}
	}

	claudeResp.Usage = &claude.ClaudeUsage{
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
	}

	if oaiResp.Usage.PromptTokensDetails != nil && (oaiResp.Usage.PromptTokensDetails.CacheCreationInputTokens > 0 || oaiResp.Usage.PromptTokensDetails.CacheReadInputTokens > 0) {
		if oaiResp.Usage.PromptTokensDetails.CacheCreationInputTokens > 0 {
			claudeResp.Usage.CacheCreationInputTokens = oaiResp.Usage.PromptTokensDetails.CacheCreationInputTokens
			claudeResp.Usage.InputTokens -= oaiResp.Usage.PromptTokensDetails.CacheCreationInputTokens
		}
		if oaiResp.Usage.PromptTokensDetails.CacheReadInputTokens > 0 {
			claudeResp.Usage.CacheReadInputTokens = oaiResp.Usage.PromptTokensDetails.CacheReadInputTokens
			claudeResp.Usage.InputTokens -= oaiResp.Usage.PromptTokensDetails.CacheReadInputTokens
		}
	}
	return nil
}

func transformRequestToClaude(ctx context.Context, oaiReq *openai.ChatCompletionRequest, claudeReq *claude.ClaudeRequest) error {
	// TODO: Implement OpenAI -> Claude request transformation
	return fmt.Errorf("OpenAI -> Claude request transformation not yet implemented")
}

func transformRequestToGemini(ctx context.Context, oaiReq *openai.ChatCompletionRequest, geminiReq *gemini.GeminiChatRequest) error {
	geminiReq.Contents = make([]gemini.GeminiChatContent, 0, len(oaiReq.Messages))

	// Generation config
	geminiReq.GenerationConfig = gemini.GeminiChatGenerationConfig{
		Temperature: func() *float64 {
			if oaiReq.Temperature == 0 {
				return nil
			}
			t := float64(oaiReq.Temperature)
			return &t
		}(),
		TopP: func() float64 {
			if oaiReq.TopP == 0 {
				return 0
			}
			return float64(oaiReq.TopP)
		}(),
		MaxOutputTokens: uint(oaiReq.MaxTokens),
		Seed: func() int64 {
			if oaiReq.Seed == nil {
				return 0
			}
			return int64(*oaiReq.Seed)
		}(),
	}

	// Safety settings - disable all
	geminiReq.SafetySettings = []gemini.GeminiChatSafetySettings{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "BLOCK_NONE"},
	}

	// Handle tools
	for _, tool := range oaiReq.Tools {
		switch tool.Function.Name {
		case "googleSearch", "google_search":
			geminiReq.Tools = append(geminiReq.Tools, gemini.GeminiChatTool{
				GoogleSearch: make(map[string]string),
			})
		case "codeExecution", "code_execution":
			geminiReq.Tools = append(geminiReq.Tools, gemini.GeminiChatTool{
				CodeExecution: make(map[string]string),
			})
		default:
			geminiReq.Tools = append(geminiReq.Tools, gemini.GeminiChatTool{
				FunctionDeclarations: tool.Function,
			})
		}
	}

	// Handle response format
	if respFormat := oaiReq.ResponseFormat; respFormat != nil && (respFormat.Type == "json_schema" || respFormat.Type == "json_object") {
		geminiReq.GenerationConfig.ResponseMimeType = "application/json"
		if respFormat.JSONSchema != nil && respFormat.JSONSchema.Schema != nil {
			geminiReq.GenerationConfig.ResponseSchema = respFormat.JSONSchema.Schema
		}
	}

	// Process messages
	toolCallIds := make(map[string]string)
	var systemContents []string

	for _, message := range oaiReq.Messages {
		switch message.Role {
		case "system", "developer":
			systemContents = append(systemContents, func() string {
				if message.Content == "" && len(message.MultiContent) > 0 {
					for _, part := range message.MultiContent {
						if part.Type == openai.ChatMessagePartTypeText {
							return part.Text
						}
					}
				}
				return message.Content
			}())
		case "tool":
			name := message.Name
			if name == "" {
				if val, exists := toolCallIds[message.ToolCallID]; exists {
					name = val
				}
			}

			var contentMap map[string]any
			if err := json.Unmarshal([]byte(message.Content), &contentMap); err != nil {
				var contentSlice []any
				if err := json.Unmarshal([]byte(message.Content), &contentSlice); err == nil {
					contentMap = map[string]any{"result": contentSlice}
				} else {
					contentMap = map[string]any{"content": message.Content}
				}
			}

			geminiReq.Contents = append(geminiReq.Contents, gemini.GeminiChatContent{
				Role: "user",
				Parts: []gemini.GeminiPart{
					{
						FunctionResponse: &gemini.FunctionResponse{
							Name:     name,
							Response: contentMap,
						},
					},
				},
			})
		case "user", "assistant":
			var parts []gemini.GeminiPart
			content := gemini.GeminiChatContent{
				Role: func() string {
					if message.Role == "assistant" {
						return "model"
					}
					return message.Role
				}(),
			}

			// Handle tool calls
			if len(message.ToolCalls) > 0 {
				for _, call := range message.ToolCalls {
					toolCall := gemini.GeminiPart{
						FunctionCall: &gemini.FunctionCall{
							FunctionName: call.Function.Name,
							Arguments: func() any {
								var args map[string]any
								if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
									return call.Function.Arguments
								}
								return args
							}(),
						},
					}
					parts = append(parts, toolCall)
					toolCallIds[call.ID] = call.Function.Name
				}
			}

			// Handle text content
			if message.Content != "" {
				parts = append(parts, gemini.GeminiPart{
					Text: message.Content,
				})
			} else {
				// Handle multi-content
				for _, ocontent := range message.MultiContent {
					switch ocontent.Type {
					case openai.ChatMessagePartTypeText:
						parts = append(parts, gemini.GeminiPart{
							Text: ocontent.Text,
						})
					case openai.ChatMessagePartTypeImageURL:
						URL := ocontent.ImageURL.URL
						if subStrs := strings.SplitN(URL, ",", 2); strings.HasPrefix(URL, "data:") && len(subStrs) == 2 {
							mediaTypePart := strings.TrimPrefix(subStrs[0], "data:")
							mediaType := strings.TrimSuffix(mediaTypePart, ";base64")

							parts = append(parts, gemini.GeminiPart{
								InlineData: &gemini.GeminiInlineData{
									MimeType: mediaType,
									Data:     subStrs[1],
								},
							})
						}
					}
				}
			}

			content.Parts = parts
			if len(content.Parts) > 0 {
				geminiReq.Contents = append(geminiReq.Contents, content)
			}
		}
	}

	// Add system instruction
	if len(systemContents) > 0 {
		geminiReq.SystemInstructions = &gemini.GeminiChatContent{
			Parts: []gemini.GeminiPart{
				{
					Text: strings.Join(systemContents, "\n"),
				},
			},
		}
	}

	return nil
}

// transformStreamResponse transforms OpenAI stream response to Claude stream response
func (t *OpenAITransformer) transformStreamResponse(ctx context.Context, src interface{}, dst interface{}) error {
	// This would handle the full stream response transformation
	return fmt.Errorf("stream response transformation not yet implemented")
}

// transformChunk transforms OpenAI chunk to other provider's chunk
func (t *OpenAITransformer) transformChunk(ctx context.Context, src interface{}, dst interface{}) error {
	oaiChunk, ok := src.(*openai.ChatCompletionStreamResponse)
	if !ok {
		return fmt.Errorf("invalid source type for OpenAI transformer")
	}

	switch dst.(type) {
	case []*claude.ClaudeResponse:
		return t.transformChunkToClaude(ctx, oaiChunk, dst.(*claude.ClaudeResponse))
	default:
		return fmt.Errorf("target type not supported for OpenAI transformer")
	}
}

func (t *OpenAITransformer) transformChunkToClaude(ctx context.Context, oaiChunk *openai.ChatCompletionStreamResponse, claudeResp *claude.ClaudeResponse) error {
	claudeResp.Id = oaiChunk.ID
	claudeResp.Model = oaiChunk.Model
	claudeResp.Type = "message"
	claudeResp.Role = "assistant"

	return nil
}

// Helper functions

func stopReasonOpenAI2Claude(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "stop_sequence":
		return "stop_sequence"
	case "max_tokens":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return reason
	}
}
