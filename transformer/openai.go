package transformer

import (
	"context"
	"fmt"

	"github.com/phosae/llms/claude"
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
	return fmt.Errorf("request transformation not yet implemented")
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
	claudeResp.Type = oaiResp.Object
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

// transformStreamResponse transforms OpenAI stream response to Claude stream response
func (t *OpenAITransformer) transformStreamResponse(ctx context.Context, src interface{}, dst interface{}) error {
	// This would handle the full stream response transformation
	return fmt.Errorf("stream response transformation not yet implemented")
}

// transformChunk transforms OpenAI chunk to Claude chunk
func (t *OpenAITransformer) transformChunk(ctx context.Context, src interface{}, dst interface{}) error {
	return fmt.Errorf("chunk transformation not yet implemented")
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
