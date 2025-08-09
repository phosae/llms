package transformer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/phosae/llms/claude"
	"github.com/phosae/llms/openai"
)

// OpenAITransformer handles OpenAI format transformations
type OpenAITransformer struct{}

// NewOpenAITransformer creates a new OpenAI transformer
func NewOpenAITransformer() *OpenAITransformer {
	return &OpenAITransformer{}
}

// GetProvider returns the provider this transformer handles
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

// ToUnified converts OpenAI request to unified format
func (t *OpenAITransformer) ToUnified(ctx context.Context, providerRequest interface{}) (*UnifiedRequest, error) {
	req, ok := providerRequest.(*openai.ChatCompletionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for OpenAI transformer")
	}

	if err := t.ValidateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	unified := &UnifiedRequest{
		Model:         req.Model,
		MaxTokens:     req.MaxTokens,
		Stream:        req.Stream,
		StopSequences: req.Stop,
		User:          req.User,
	}

	// Handle optional fields with type conversion
	if req.Temperature != 0 {
		temp := float64(req.Temperature)
		unified.Temperature = &temp
	}
	if req.TopP != 0 {
		topP := float64(req.TopP)
		unified.TopP = &topP
	}
	if req.FrequencyPenalty != 0 {
		fp := float64(req.FrequencyPenalty)
		unified.FrequencyPenalty = &fp
	}
	if req.PresencePenalty != 0 {
		pp := float64(req.PresencePenalty)
		unified.PresencePenalty = &pp
	}

	// Handle max completion tokens (prioritize over deprecated MaxTokens)
	if req.MaxCompletionTokens > 0 {
		unified.MaxTokens = req.MaxCompletionTokens
	}

	// Handle seed
	if req.Seed != nil {
		seed := int64(*req.Seed)
		unified.Seed = &seed
	}

	// Convert messages
	for _, msg := range req.Messages {
		unifiedMsg := UnifiedMessage{
			Role: msg.Role,
			Name: msg.Name,
		}

		// Handle content
		if msg.Content != "" {
			unifiedMsg.Content = msg.Content
		} else if len(msg.MultiContent) > 0 {
			// Convert multipart content
			for _, part := range msg.MultiContent {
				unifiedPart := UnifiedMessagePart{
					Type: string(part.Type),
					Text: part.Text,
				}

				if part.ImageURL != nil {
					unifiedPart.ImageURL = &UnifiedImageURL{
						URL:    part.ImageURL.URL,
						Detail: string(part.ImageURL.Detail),
					}
				}

				unifiedMsg.Parts = append(unifiedMsg.Parts, unifiedPart)
			}
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				}

				unifiedMsg.ToolCalls = append(unifiedMsg.ToolCalls, UnifiedToolCall{
					ID:        toolCall.ID,
					Type:      string(toolCall.Type),
					Name:      toolCall.Function.Name,
					Arguments: args,
				})
			}
		}

		// Handle tool call ID
		if msg.ToolCallID != "" {
			unifiedMsg.ToolCallID = msg.ToolCallID
		}

		unified.Messages = append(unified.Messages, unifiedMsg)
	}

	// Convert tools
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			if tool.Function != nil {
				unifiedTool := UnifiedTool{
					Type:        string(tool.Type),
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  convertAnyToMap(tool.Function.Parameters),
				}
				unified.Tools = append(unified.Tools, unifiedTool)
			}
		}
	}

	// Handle tool choice
	if req.ToolChoice != nil {
		switch tc := req.ToolChoice.(type) {
		case string:
			unified.ToolChoice = tc
		case openai.ToolChoice:
			unified.ToolChoice = tc.Function.Name
		}
	}

	// Handle response format
	if req.ResponseFormat != nil {
		unified.ResponseFormat = &UnifiedResponseFormat{
			Type: string(req.ResponseFormat.Type),
		}
		if req.ResponseFormat.JSONSchema != nil {
			unified.ResponseFormat.Schema = map[string]interface{}{
				"name":        req.ResponseFormat.JSONSchema.Name,
				"description": req.ResponseFormat.JSONSchema.Description,
				"strict":      req.ResponseFormat.JSONSchema.Strict,
			}
		}
	}

	return unified, nil
}

// FromUnified converts unified request to OpenAI format
func (t *OpenAITransformer) FromUnified(ctx context.Context, unifiedRequest *UnifiedRequest) (interface{}, error) {
	req := &openai.ChatCompletionRequest{
		Model:     unifiedRequest.Model,
		MaxTokens: unifiedRequest.MaxTokens,
		Stream:    unifiedRequest.Stream,
		Stop:      unifiedRequest.StopSequences,
		User:      unifiedRequest.User,
	}

	// Handle optional fields
	if unifiedRequest.Temperature != nil {
		req.Temperature = float32(*unifiedRequest.Temperature)
	}
	if unifiedRequest.TopP != nil {
		req.TopP = float32(*unifiedRequest.TopP)
	}
	if unifiedRequest.FrequencyPenalty != nil {
		req.FrequencyPenalty = float32(*unifiedRequest.FrequencyPenalty)
	}
	if unifiedRequest.PresencePenalty != nil {
		req.PresencePenalty = float32(*unifiedRequest.PresencePenalty)
	}
	if unifiedRequest.Seed != nil {
		seed := int(*unifiedRequest.Seed)
		req.Seed = &seed
	}

	// Convert messages
	for _, unifiedMsg := range unifiedRequest.Messages {
		msg := openai.ChatCompletionMessage{
			Role: unifiedMsg.Role,
			Name: unifiedMsg.Name,
		}

		// Handle content
		if unifiedMsg.Content != "" {
			msg.Content = unifiedMsg.Content
		} else if len(unifiedMsg.Parts) > 0 {
			// Convert multipart content
			for _, part := range unifiedMsg.Parts {
				openaiPart := openai.ChatMessagePart{
					Type: openai.ChatMessagePartType(part.Type),
					Text: part.Text,
				}

				if part.ImageURL != nil {
					openaiPart.ImageURL = &openai.ChatMessageImageURL{
						URL:    part.ImageURL.URL,
						Detail: openai.ImageURLDetail(part.ImageURL.Detail),
					}
				}

				msg.MultiContent = append(msg.MultiContent, openaiPart)
			}
		}

		// Handle tool calls
		if len(unifiedMsg.ToolCalls) > 0 {
			for _, toolCall := range unifiedMsg.ToolCalls {
				args, _ := json.Marshal(toolCall.Arguments)

				msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
					ID:   toolCall.ID,
					Type: openai.ToolType(toolCall.Type),
					Function: openai.FunctionCall{
						Name:      toolCall.Name,
						Arguments: string(args),
					},
				})
			}
		}

		// Handle tool call ID
		if unifiedMsg.ToolCallID != "" {
			msg.ToolCallID = unifiedMsg.ToolCallID
		}

		req.Messages = append(req.Messages, msg)
	}

	// Convert tools
	if len(unifiedRequest.Tools) > 0 {
		for _, unifiedTool := range unifiedRequest.Tools {
			tool := openai.Tool{
				Type: openai.ToolType(unifiedTool.Type),
				Function: &openai.FunctionDefinition{
					Name:        unifiedTool.Name,
					Description: unifiedTool.Description,
					Parameters:  unifiedTool.Parameters,
				},
			}
			req.Tools = append(req.Tools, tool)
		}
	}

	// Handle tool choice
	if unifiedRequest.ToolChoice != "" {
		if unifiedRequest.ToolChoice == "auto" || unifiedRequest.ToolChoice == "none" {
			req.ToolChoice = unifiedRequest.ToolChoice
		} else {
			// Specific tool choice
			req.ToolChoice = openai.ToolChoice{
				Type: openai.ToolTypeFunction,
				Function: openai.ToolFunction{
					Name: unifiedRequest.ToolChoice,
				},
			}
		}
	}

	// Handle response format
	if unifiedRequest.ResponseFormat != nil {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatType(unifiedRequest.ResponseFormat.Type),
		}
	}

	return req, nil
}

// ResponseToUnified converts OpenAI response to unified format
func (t *OpenAITransformer) ResponseToUnified(ctx context.Context, providerResponse interface{}) (*UnifiedResponse, error) {
	resp, ok := providerResponse.(*openai.ChatCompletionResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type for OpenAI transformer")
	}

	unified := &UnifiedResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		Provider:          ProviderOpenAI,
		SystemFingerprint: resp.SystemFingerprint,
	}

	// Convert usage
	unified.Usage = &UnifiedUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	// Convert choices
	for _, choice := range resp.Choices {
		unifiedChoice := UnifiedChoice{
			Index:        choice.Index,
			FinishReason: string(choice.FinishReason),
		}

		// Convert message
		unifiedChoice.Message = UnifiedMessage{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
			Name:    choice.Message.Name,
		}

		// Handle multipart content
		if len(choice.Message.MultiContent) > 0 {
			for _, part := range choice.Message.MultiContent {
				unifiedPart := UnifiedMessagePart{
					Type: string(part.Type),
					Text: part.Text,
				}
				if part.ImageURL != nil {
					unifiedPart.ImageURL = &UnifiedImageURL{
						URL:    part.ImageURL.URL,
						Detail: string(part.ImageURL.Detail),
					}
				}
				unifiedChoice.Message.Parts = append(unifiedChoice.Message.Parts, unifiedPart)
			}
		}

		// Handle tool calls
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				}

				unifiedChoice.Message.ToolCalls = append(unifiedChoice.Message.ToolCalls, UnifiedToolCall{
					ID:        toolCall.ID,
					Type:      string(toolCall.Type),
					Name:      toolCall.Function.Name,
					Arguments: args,
				})
			}
		}

		// Handle log probabilities
		if choice.LogProbs != nil && len(choice.LogProbs.Content) > 0 {
			unifiedChoice.LogProbs = &UnifiedLogProbs{}
			for _, logProb := range choice.LogProbs.Content {
				unifiedChoice.LogProbs.Content = append(unifiedChoice.LogProbs.Content, UnifiedLogProb{
					Token:   logProb.Token,
					LogProb: logProb.LogProb,
					Bytes:   logProb.Bytes,
				})
			}
		}

		unified.Choices = append(unified.Choices, unifiedChoice)
	}

	return unified, nil
}

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

func (t *OpenAITransformer) ResponseToClaude(ctx context.Context, srcResp interface{}, dstResp interface{}) error {
	oaiResp, ok := srcResp.(*openai.ChatCompletionResponse)
	if !ok {
		return fmt.Errorf("invalid response type for OpenAI transformer")
	}

	claudeResp, ok := dstResp.(*claude.ClaudeResponse)
	if !ok {
		return fmt.Errorf("invalid response type for Claude transformer")
	}

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

func (t *OpenAITransformer) StreamResponseToClaude(ctx context.Context, srcResp interface{}, dstResp interface{}) error {
	oaiResp, ok := srcResp.(*openai.ChatCompletionStreamResponse)
	if !ok {
		return fmt.Errorf("invalid response type for OpenAI transformer")
	}

	claudeResp, ok := dstResp.(*claude.ClaudeResponse)
	if !ok {
		return fmt.Errorf("invalid response type for Claude transformer")
	}

	fmt.Println("oaiResp is", oaiResp)
	fmt.Println("claudeResp is", claudeResp)
	return nil
}

// ResponseFromUnified converts unified response to OpenAI format
func (t *OpenAITransformer) ResponseFromUnified(ctx context.Context, unifiedResponse *UnifiedResponse) (interface{}, error) {
	resp := &openai.ChatCompletionResponse{
		ID:                unifiedResponse.ID,
		Object:            unifiedResponse.Object,
		Created:           unifiedResponse.Created,
		Model:             unifiedResponse.Model,
		SystemFingerprint: unifiedResponse.SystemFingerprint,
	}

	// Convert usage
	if unifiedResponse.Usage != nil {
		resp.Usage = openai.Usage{
			PromptTokens:     unifiedResponse.Usage.PromptTokens,
			CompletionTokens: unifiedResponse.Usage.CompletionTokens,
			TotalTokens:      unifiedResponse.Usage.TotalTokens,
		}
	}

	// Convert choices
	for _, unifiedChoice := range unifiedResponse.Choices {
		choice := openai.ChatCompletionChoice{
			Index:        unifiedChoice.Index,
			FinishReason: openai.FinishReason(unifiedChoice.FinishReason),
		}

		// Convert message
		choice.Message = openai.ChatCompletionMessage{
			Role:    unifiedChoice.Message.Role,
			Content: unifiedChoice.Message.Content,
			Name:    unifiedChoice.Message.Name,
		}

		// Handle multipart content
		if len(unifiedChoice.Message.Parts) > 0 {
			for _, part := range unifiedChoice.Message.Parts {
				openaiPart := openai.ChatMessagePart{
					Type: openai.ChatMessagePartType(part.Type),
					Text: part.Text,
				}
				if part.ImageURL != nil {
					openaiPart.ImageURL = &openai.ChatMessageImageURL{
						URL:    part.ImageURL.URL,
						Detail: openai.ImageURLDetail(part.ImageURL.Detail),
					}
				}
				choice.Message.MultiContent = append(choice.Message.MultiContent, openaiPart)
			}
		}

		// Handle tool calls
		if len(unifiedChoice.Message.ToolCalls) > 0 {
			for _, toolCall := range unifiedChoice.Message.ToolCalls {
				args, _ := json.Marshal(toolCall.Arguments)

				choice.Message.ToolCalls = append(choice.Message.ToolCalls, openai.ToolCall{
					ID:   toolCall.ID,
					Type: openai.ToolType(toolCall.Type),
					Function: openai.FunctionCall{
						Name:      toolCall.Name,
						Arguments: string(args),
					},
				})
			}
		}

		resp.Choices = append(resp.Choices, choice)
	}

	return resp, nil
}

// convertAnyToMap converts any type to map[string]interface{}
func convertAnyToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}

	// If already a map, return it
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}

	// Try to convert via JSON marshaling/unmarshaling
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}
