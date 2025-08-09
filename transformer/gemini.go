package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/phosae/llms/gemini"
	"github.com/phosae/llms/openai"
)

// GeminiTransformer handles direct Gemini to OpenAI transformations
type GeminiTransformer struct{}

// NewGeminiTransformer creates a new Gemini to OpenAI transformer
func NewGeminiTransformer() *GeminiTransformer {
	return &GeminiTransformer{}
}

// GetProvider returns the source provider (Gemini)
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

// Do performs the transformation based on the type
func (t *GeminiTransformer) Do(ctx context.Context, typ TransformerType, src interface{}, dst interface{}) error {
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

func (t *GeminiTransformer) transformRequest(ctx context.Context, src interface{}, dst interface{}) error {
	return fmt.Errorf("request transformation not yet implemented")
}

func (t *GeminiTransformer) transformResponse(ctx context.Context, src interface{}, dst interface{}) error {
	geminiResp, ok := src.(*gemini.GeminiChatResponse)
	if !ok {
		return fmt.Errorf("invalid source type for Gemini transformer")
	}

	switch target := dst.(type) {
	case *openai.ChatCompletionResponse:
		return transformGeminiResponseToOpenAI(ctx, geminiResp, target)
	default:
		return fmt.Errorf("target type not supported for Gemini transformer")
	}
}

func transformGeminiResponseToOpenAI(ctx context.Context, geminiResp *gemini.GeminiChatResponse, oaiResp *openai.ChatCompletionResponse) error {
	oaiResp.Object = "chat.completion"
	oaiResp.Created = time.Now().Unix()
	oaiResp.Choices = make([]openai.ChatCompletionChoice, 0, len(geminiResp.Candidates))

	isToolCall := false
	for candidateIndex, candidate := range geminiResp.Candidates {
		choice := openai.ChatCompletionChoice{
			Index: int(candidate.Index),
			Message: openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: "",
			},
			FinishReason: openai.FinishReasonStop,
		}

		if len(candidate.Content.Parts) > 0 {
			var texts []string
			var toolCalls []openai.ToolCall

			for i, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					choice.FinishReason = openai.FinishReasonToolCalls
					if call, err := parseGeminiToolCall(&part); err != nil {
						return fmt.Errorf("failed to parse tool call candidates[%d].parts[%d]: %v", candidateIndex, i, err)
					} else {
						toolCalls = append(toolCalls, *call)
					}
				} else if part.Thought {
					choice.Message.ReasoningContent = part.Text
				} else {
					if part.ExecutableCode != nil {
						texts = append(texts, "```"+part.ExecutableCode.Language+"\n"+part.ExecutableCode.Code+"\n```")
					} else if part.CodeExecutionResult != nil {
						texts = append(texts, "```output\n"+part.CodeExecutionResult.Output+"\n```")
					} else {
						// Filter out empty lines
						if part.Text != "\n" && part.Text != "" {
							texts = append(texts, part.Text)
						}
					}
				}
			}

			if len(toolCalls) > 0 {
				choice.Message.ToolCalls = toolCalls
				isToolCall = true
			}
			choice.Message.Content = strings.Join(texts, "\n")
		}

		if candidate.FinishReason != nil {
			switch *candidate.FinishReason {
			case "STOP":
				choice.FinishReason = openai.FinishReasonStop
			case "MAX_TOKENS":
				choice.FinishReason = openai.FinishReasonLength
			default:
				choice.FinishReason = openai.FinishReasonContentFilter
			}
		}

		if isToolCall {
			choice.FinishReason = openai.FinishReasonToolCalls
		}

		oaiResp.Choices = append(oaiResp.Choices, choice)
	}

	// Convert usage metadata
	if geminiResp.UsageMetadata.TotalTokenCount > 0 {
		oaiResp.Usage = openai.Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}

		if geminiResp.UsageMetadata.ThoughtsTokenCount > 0 {
			oaiResp.Usage.CompletionTokensDetails = &openai.CompletionTokensDetails{
				ReasoningTokens: geminiResp.UsageMetadata.ThoughtsTokenCount,
			}
		}

		if geminiResp.UsageMetadata.CachedContentTokenCount > 0 {
			oaiResp.Usage.PromptTokensDetails = &openai.PromptTokensDetails{
				CachedTokens: geminiResp.UsageMetadata.CachedContentTokenCount,
			}
		}
	}

	return nil
}

func (t *GeminiTransformer) transformStreamResponse(ctx context.Context, src interface{}, dst interface{}) error {
	return fmt.Errorf("stream response transformation not yet implemented")
}

func (t *GeminiTransformer) transformChunk(ctx context.Context, src interface{}, dst interface{}) error {
	geminiChunk, ok := src.(*gemini.GeminiChatResponse)
	if !ok {
		return fmt.Errorf("invalid source type for Gemini transformer")
	}

	switch target := dst.(type) {
	case *openai.ChatCompletionStreamResponse:
		return transformGeminiChunkToOpenAI(ctx, geminiChunk, target)
	default:
		return fmt.Errorf("target type not supported for Gemini transformer")
	}
}

func transformGeminiChunkToOpenAI(ctx context.Context, geminiChunk *gemini.GeminiChatResponse, oaiChunk *openai.ChatCompletionStreamResponse) error {
	oaiChunk.Object = "chat.completion.chunk"
	oaiChunk.Choices = make([]openai.ChatCompletionStreamChoice, 0, len(geminiChunk.Candidates))

	for candidateIndex, candidate := range geminiChunk.Candidates {
		choice := openai.ChatCompletionStreamChoice{
			Index: int(candidate.Index),
			Delta: openai.ChatCompletionStreamChoiceDelta{
				Role: "assistant",
			},
		}

		var texts []string
		isTools := false
		isThought := false

		if candidate.FinishReason != nil {
			switch *candidate.FinishReason {
			case "STOP":
				choice.FinishReason = "stop"
			case "MAX_TOKENS":
				choice.FinishReason = "length"
			default:
				choice.FinishReason = "content_filter"
			}
		}

		for i, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				isTools = true
				if call, err := parseGeminiToolCall(&part); err == nil {
					call.Index = func() *int {
						idx := len(choice.Delta.ToolCalls)
						return &idx
					}()
					choice.Delta.ToolCalls = append(choice.Delta.ToolCalls, *call)
				} else {
					return fmt.Errorf("failed to parse tool call candidates[%d].parts[%d]: %v", candidateIndex, i, err)
				}
			} else if part.Thought {
				isThought = true
				texts = append(texts, part.Text)
			} else {
				if part.ExecutableCode != nil {
					texts = append(texts, "```"+part.ExecutableCode.Language+"\n"+part.ExecutableCode.Code+"\n```\n")
				} else if part.CodeExecutionResult != nil {
					texts = append(texts, "```output\n"+part.CodeExecutionResult.Output+"\n```\n")
				} else {
					if part.Text != "\n" && part.Text != "" {
						texts = append(texts, part.Text)
					}
				}
			}
		}

		if isThought {
			choice.Delta.ReasoningContent = strings.Join(texts, "\n")
		} else {
			choice.Delta.Content = strings.Join(texts, "\n")
		}

		if isTools {
			choice.FinishReason = "tool_calls"
		}

		oaiChunk.Choices = append(oaiChunk.Choices, choice)
	}

	return nil
}

// Helper functions
func parseGeminiToolCall(part *gemini.GeminiPart) (*openai.ToolCall, error) {
	var argsBytes []byte
	var err error

	if result, ok := part.FunctionCall.Arguments.(map[string]interface{}); ok {
		argsBytes, err = json.Marshal(result)
	} else {
		argsBytes, err = json.Marshal(part.FunctionCall.Arguments)
	}

	if err != nil {
		return nil, err
	}

	return &openai.ToolCall{
		ID:   fmt.Sprintf("call_%s", generateUUID()),
		Type: "function",
		Function: openai.FunctionCall{
			Arguments: string(argsBytes),
			Name:      part.FunctionCall.FunctionName,
		},
	}, nil
}

// Simple UUID generator (simplified version)
func generateUUID() string {
	// This is a simplified UUID generator for demo purposes
	// In production, use a proper UUID library
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
