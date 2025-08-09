package transformer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/phosae/llms/claude"
	"github.com/phosae/llms/common"
	"github.com/phosae/llms/gemini"
	"github.com/phosae/llms/openai"
)

// ClaudeTransformer handles direct Claude to OpenAI transformations
type ClaudeTransformer struct{}

// NewClaudeTransformer creates a new Claude to OpenAI transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// GetProvider returns the source provider (Claude)
func (t *ClaudeTransformer) GetProvider() Provider {
	return ProviderClaude
}

// ValidateRequest validates the Claude request
func (t *ClaudeTransformer) ValidateRequest(ctx context.Context, request interface{}) error {
	req, ok := request.(*claude.ClaudeRequest)
	if !ok {
		return fmt.Errorf("invalid request type for Claude transformer")
	}

	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	if req.MaxTokens == 0 {
		return fmt.Errorf("max_tokens is required")
	}

	return nil
}

// Do performs the transformation based on the type
func (t *ClaudeTransformer) Do(ctx context.Context, typ TransformerType, src interface{}, dst interface{}) error {
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

// transformRequest transforms Claude request to OpenAI request
func (t *ClaudeTransformer) transformRequest(ctx context.Context, src interface{}, dst interface{}) error {
	claudeReq, ok := src.(*claude.ClaudeRequest)
	if !ok {
		return fmt.Errorf("invalid source type for Claude request transformer")
	}

	switch dst.(type) {
	case *openai.ChatCompletionRequest:
		return transformRequestToOpenAI(ctx, claudeReq, dst.(*openai.ChatCompletionRequest))
	case *claude.ClaudeRequest:
		return nil
	case *gemini.GeminiChatRequest:
		return fmt.Errorf("gemini is not supported")
	default:
		return fmt.Errorf("invalid target type for Claude transformer")
	}
}

func transformRequestToOpenAI(ctx context.Context, claudeReq *claude.ClaudeRequest, oaiReq *openai.ChatCompletionRequest) error {
	oaiReq.Model = claudeReq.Model
	oaiReq.MaxTokens = int(claudeReq.MaxTokens)
	oaiReq.Temperature = func() float32 {
		if claudeReq.Temperature == nil {
			return 0
		}
		return float32(*claudeReq.Temperature)
	}()
	oaiReq.TopP = float32(claudeReq.TopP)
	oaiReq.Stream = claudeReq.Stream
	oaiReq.Stop = claudeReq.StopSequences

	if claudeReq.Thinking != nil && claudeReq.Thinking.Type == "enabled" {
		budgetTokens := claudeReq.Thinking.GetBudgetTokens()
		if budgetTokens > 0 {
			if budgetTokens < 1024 {
				oaiReq.ReasoningEffort = "low"
			} else if budgetTokens < 2048 {
				oaiReq.ReasoningEffort = "medium"
			} else {
				oaiReq.ReasoningEffort = "high"
			}
		}
	}

	tools, _ := common.Any2Type[[]claude.Tool](claudeReq.Tools)
	openAITools := make([]openai.Tool, 0)
	for _, claudeTool := range tools {
		openAITools = append(openAITools, openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        claudeTool.Name,
				Description: claudeTool.Description,
				Parameters:  claudeTool.InputSchema,
			},
		})
	}
	oaiReq.Tools = openAITools

	oaiMessages := make([]openai.ChatCompletionMessage, 0)
	if claudeReq.System != nil {
		if claudeReq.IsStringSystem() && claudeReq.GetStringSystem() != "" {
			oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
				Role:    "system",
				Content: claudeReq.GetStringSystem(),
			})
		} else {
			systems := claudeReq.ParseSystem()
			if len(systems) > 0 {
				oaiSysMessage := openai.ChatCompletionMessage{Role: "system"}
				for _, system := range systems {
					oaiSysMessage.MultiContent = append(oaiSysMessage.MultiContent, openai.ChatMessagePart{
						Type:         "text",
						Text:         system.GetText(),
						CacheControl: system.CacheControl,
					})
				}
				oaiMessages = append(oaiMessages, oaiSysMessage)
			}
		}
	}

	for _, claudeMessage := range claudeReq.Messages {
		openAIMessage := openai.ChatCompletionMessage{
			Role: claudeMessage.Role,
		}

		if claudeMessage.IsStringContent() {
			openAIMessage.Content = claudeMessage.GetStringContent()
		} else {
			contents, err := claudeMessage.ParseContent()
			if err != nil {
				return err
			}
			parts := make([]openai.ChatMessagePart, 0, len(contents))

			for _, content := range contents {
				switch content.Type {
				case "text":
					parts = append(parts, openai.ChatMessagePart{
						Type:         "text",
						Text:         content.GetText(),
						CacheControl: content.CacheControl,
					})
				case "image":
					var imageData string
					switch content.Source.Type {
					case "base64":
						imageData = fmt.Sprintf("data:%s;base64,%s", content.Source.MediaType, content.Source.Data)
					case "url":
						imageData = content.Source.Url
					}
					parts = append(parts, openai.ChatMessagePart{
						Type: "image_url",
						ImageURL: &openai.ChatMessageImageURL{
							URL: imageData,
						},
					})
				case "tool_use":
					openAIMessage.ToolCalls = append(openAIMessage.ToolCalls, openai.ToolCall{
						ID:   content.Id,
						Type: "function",
						Function: openai.FunctionCall{
							Name:      content.Name,
							Arguments: toJSONString(content.Input),
						},
					})
				case "tool_result":
					// Add tool result as a separate message
					oaiToolMessage := openai.ChatCompletionMessage{
						Role:       "tool",
						Name:       content.Name,
						ToolCallID: content.ToolUseId,
					}
					if content.IsStringContent() {
						oaiToolMessage.Content = content.GetStringContent()
					} else {
						mContents := content.ParseMediaContent()
						json, _ := json.Marshal(mContents)
						oaiToolMessage.Content = string(json)
					}
					oaiMessages = append(oaiMessages, oaiToolMessage)
				}
			}
			openAIMessage.MultiContent = parts
		}

		if len(openAIMessage.Content) > 0 || len(openAIMessage.MultiContent) > 0 || len(openAIMessage.ToolCalls) > 0 {
			oaiMessages = append(oaiMessages, openAIMessage)
		}
	}

	oaiReq.Messages = oaiMessages
	return nil
}

// transformResponse transforms Claude response to OpenAI response
func (t *ClaudeTransformer) transformResponse(ctx context.Context, src interface{}, dst interface{}) error {
	return fmt.Errorf("response transformation not yet implemented")
}

// transformStreamResponse transforms Claude stream response to OpenAI stream response
func (t *ClaudeTransformer) transformStreamResponse(ctx context.Context, src interface{}, dst interface{}) error {
	// This would handle the full stream response transformation
	return fmt.Errorf("stream response transformation not yet implemented")
}

// transformChunk transforms Claude chunk to OpenAI chunk
func (t *ClaudeTransformer) transformChunk(ctx context.Context, src interface{}, dst interface{}) error {
	return fmt.Errorf("chunk transformation not yet implemented")
}

// helper functions

func toJSONString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
