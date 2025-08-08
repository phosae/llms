package transformer

import (
	"context"
	"fmt"

	"github.com/phosae/llms/claude"
)

// ClaudeTransformer handles Claude format transformations
type ClaudeTransformer struct{}

// NewClaudeTransformer creates a new Claude transformer
func NewClaudeTransformer() *ClaudeTransformer {
	return &ClaudeTransformer{}
}

// GetProvider returns the provider this transformer handles
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

	if len(req.Messages) == 0 && req.Prompt == "" {
		return fmt.Errorf("either messages or prompt must be provided")
	}

	if req.MaxTokens == 0 {
		return fmt.Errorf("max_tokens is required")
	}

	return nil
}

// ToUnified converts Claude request to unified format
func (t *ClaudeTransformer) ToUnified(ctx context.Context, providerRequest interface{}) (*UnifiedRequest, error) {
	req, ok := providerRequest.(*claude.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Claude transformer")
	}

	if err := t.ValidateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	unified := &UnifiedRequest{
		Model:  req.Model,
		Stream: req.Stream,
		TopP:   &req.TopP,
	}

	// Handle temperature
	if req.Temperature != nil {
		unified.Temperature = req.Temperature
	}

	// Handle stop sequences
	if len(req.StopSequences) > 0 {
		unified.StopSequences = req.StopSequences
	}

	// Handle system prompt
	if req.System != nil {
		if req.IsStringSystem() {
			unified.SystemPrompt = req.GetStringSystem()
		} else {
			// Handle complex system content
			systemParts := req.ParseSystem()
			unified.SystemPrompt = t.extractTextFromMediaMessages(systemParts)
		}
	}

	// Handle legacy prompt format
	if req.Prompt != "" {
		unified.Messages = append(unified.Messages, UnifiedMessage{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		unifiedMsg := UnifiedMessage{
			Role: msg.Role,
		}

		// Handle string content
		if msg.IsStringContent() {
			unifiedMsg.Content = msg.GetStringContent()
		} else {
			// Handle complex content
			parts, err := msg.ParseContent()
			if err == nil {
				textContent := t.extractTextFromMediaMessages(parts)
				if textContent != "" {
					unifiedMsg.Content = textContent
				}

				// Convert media parts
				for _, part := range parts {
					unifiedPart := t.convertClaudePartToUnified(part)
					if unifiedPart != nil {
						unifiedMsg.Parts = append(unifiedMsg.Parts, *unifiedPart)
					}

					// Handle tool calls
					if part.Type == "tool_use" {
						args := convertAnyToMap(part.Input)
						unifiedMsg.ToolCalls = append(unifiedMsg.ToolCalls, UnifiedToolCall{
							ID:        part.Id,
							Type:      "function",
							Name:      part.Name,
							Arguments: args,
						})
					}

					// Handle tool results
					if part.Type == "tool_result" {
						unifiedMsg.ToolCallID = part.ToolUseId
						if content, ok := part.Content.(string); ok {
							unifiedMsg.Content = content
						}
					}
				}
			}
		}

		unified.Messages = append(unified.Messages, unifiedMsg)
	}

	// Convert tools
	if req.Tools != nil {
		if tools := req.GetTools(); tools != nil {
			normalTools, _ := claude.ProcessTools(tools)
			for _, tool := range normalTools {
				unifiedTool := UnifiedTool{
					Type:        "function",
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
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
		case claude.ClaudeToolChoice:
			if tc.Type == "tool" {
				unified.ToolChoice = tc.Name
			} else {
				unified.ToolChoice = tc.Type
			}
		}
	}

	return unified, nil
}

// FromUnified converts unified request to Claude format
func (t *ClaudeTransformer) FromUnified(ctx context.Context, unifiedRequest *UnifiedRequest) (interface{}, error) {
	req := &claude.ClaudeRequest{
		Model:     unifiedRequest.Model,
		MaxTokens: uint(unifiedRequest.MaxTokens),
		Stream:    unifiedRequest.Stream,
	}

	// Handle optional fields
	if unifiedRequest.Temperature != nil {
		req.Temperature = unifiedRequest.Temperature
	}
	if unifiedRequest.TopP != nil {
		req.TopP = *unifiedRequest.TopP
	}
	if len(unifiedRequest.StopSequences) > 0 {
		req.StopSequences = unifiedRequest.StopSequences
	}

	// Handle system prompt
	if unifiedRequest.SystemPrompt != "" {
		req.SetStringSystem(unifiedRequest.SystemPrompt)
	}

	// Convert messages
	for _, unifiedMsg := range unifiedRequest.Messages {
		msg := claude.ClaudeMessage{
			Role: unifiedMsg.Role,
		}

		// Handle simple text content
		if unifiedMsg.Content != "" && len(unifiedMsg.Parts) == 0 && len(unifiedMsg.ToolCalls) == 0 {
			msg.SetStringContent(unifiedMsg.Content)
		} else {
			// Handle complex content
			var parts []claude.ClaudeMediaMessage

			// Add text content
			if unifiedMsg.Content != "" {
				parts = append(parts, claude.ClaudeMediaMessage{
					Type: "text",
					Text: &unifiedMsg.Content,
				})
			}

			// Add media parts
			for _, part := range unifiedMsg.Parts {
				claudePart := t.convertUnifiedPartToClaude(part)
				if claudePart != nil {
					parts = append(parts, *claudePart)
				}
			}

			// Add tool calls
			for _, toolCall := range unifiedMsg.ToolCalls {
				parts = append(parts, claude.ClaudeMediaMessage{
					Type:  "tool_use",
					Id:    toolCall.ID,
					Name:  toolCall.Name,
					Input: toolCall.Arguments,
				})
			}

			// Handle tool results
			if unifiedMsg.ToolCallID != "" {
				parts = append(parts, claude.ClaudeMediaMessage{
					Type:      "tool_result",
					ToolUseId: unifiedMsg.ToolCallID,
					Content:   unifiedMsg.Content,
				})
			}

			msg.Content = parts
		}

		req.Messages = append(req.Messages, msg)
	}

	// Convert tools
	if len(unifiedRequest.Tools) > 0 {
		for _, unifiedTool := range unifiedRequest.Tools {
			if unifiedTool.Type == "function" {
				tool := &claude.Tool{
					Name:        unifiedTool.Name,
					Description: unifiedTool.Description,
					InputSchema: unifiedTool.Parameters,
				}
				req.AddTool(tool)
			}
		}
	}

	// Handle tool choice
	if unifiedRequest.ToolChoice != "" {
		if unifiedRequest.ToolChoice == "auto" || unifiedRequest.ToolChoice == "any" {
			req.ToolChoice = &claude.ClaudeToolChoice{
				Type: unifiedRequest.ToolChoice,
			}
		} else {
			// Specific tool choice
			req.ToolChoice = &claude.ClaudeToolChoice{
				Type: "tool",
				Name: unifiedRequest.ToolChoice,
			}
		}
	}

	return req, nil
}

// ResponseToUnified converts Claude response to unified format
func (t *ClaudeTransformer) ResponseToUnified(ctx context.Context, providerResponse interface{}) (*UnifiedResponse, error) {
	resp, ok := providerResponse.(*claude.ClaudeResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type for Claude transformer")
	}

	unified := &UnifiedResponse{
		ID:       resp.Id,
		Object:   "chat.completion",
		Model:    resp.Model,
		Provider: ProviderClaude,
	}

	// Handle error
	if resp.Error != nil {
		unified.Error = &UnifiedError{
			Type:    resp.Error.Type,
			Message: resp.Error.Message,
		}
		return unified, nil
	}

	// Convert usage
	if resp.Usage != nil {
		unified.Usage = &UnifiedUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CacheReadTokens:  resp.Usage.CacheReadInputTokens,
			CacheWriteTokens: resp.Usage.CacheCreationInputTokens,
		}
	}

	// Handle legacy completion format
	if resp.Completion != "" {
		unified.Choices = append(unified.Choices, UnifiedChoice{
			Index: 0,
			Message: UnifiedMessage{
				Role:    "assistant",
				Content: resp.Completion,
			},
			FinishReason: resp.StopReason,
		})
		return unified, nil
	}

	// Convert content to choices
	if len(resp.Content) > 0 {
		unifiedChoice := UnifiedChoice{
			Index:        0,
			FinishReason: resp.StopReason,
		}

		unifiedChoice.Message = UnifiedMessage{
			Role: resp.Role,
		}

		// Extract text content
		var textContent string
		for _, content := range resp.Content {
			if content.Type == "text" && content.Text != nil {
				textContent += content.GetText()
			}
		}
		if textContent != "" {
			unifiedChoice.Message.Content = textContent
		}

		// Handle non-text parts and tool calls
		for _, content := range resp.Content {
			if content.Type != "text" {
				unifiedPart := t.convertClaudePartToUnified(content)
				if unifiedPart != nil {
					unifiedChoice.Message.Parts = append(unifiedChoice.Message.Parts, *unifiedPart)
				}
			}

			// Handle tool calls
			if content.Type == "tool_use" {
				args := convertAnyToMap(content.Input)
				unifiedChoice.Message.ToolCalls = append(unifiedChoice.Message.ToolCalls, UnifiedToolCall{
					ID:        content.Id,
					Type:      "function",
					Name:      content.Name,
					Arguments: args,
				})
			}
		}

		unified.Choices = append(unified.Choices, unifiedChoice)
	}

	return unified, nil
}

// ResponseFromUnified converts unified response to Claude format
func (t *ClaudeTransformer) ResponseFromUnified(ctx context.Context, unifiedResponse *UnifiedResponse) (interface{}, error) {
	resp := &claude.ClaudeResponse{
		Id:    unifiedResponse.ID,
		Type:  "message",
		Model: unifiedResponse.Model,
	}

	// Handle error
	if unifiedResponse.Error != nil {
		resp.Error = &claude.ClaudeError{
			Type:    unifiedResponse.Error.Type,
			Message: unifiedResponse.Error.Message,
		}
		return resp, nil
	}

	// Convert usage
	if unifiedResponse.Usage != nil {
		resp.Usage = &claude.ClaudeUsage{
			InputTokens:              unifiedResponse.Usage.PromptTokens,
			OutputTokens:             unifiedResponse.Usage.CompletionTokens,
			CacheReadInputTokens:     unifiedResponse.Usage.CacheReadTokens,
			CacheCreationInputTokens: unifiedResponse.Usage.CacheWriteTokens,
		}
	}

	// Convert choices to content
	if len(unifiedResponse.Choices) > 0 {
		choice := unifiedResponse.Choices[0] // Claude typically has one choice
		resp.Role = choice.Message.Role
		resp.StopReason = choice.FinishReason

		// Handle text content
		if choice.Message.Content != "" {
			text := choice.Message.Content
			resp.Content = append(resp.Content, claude.ClaudeMediaMessage{
				Type: "text",
				Text: &text,
			})
		}

		// Handle multipart content
		for _, part := range choice.Message.Parts {
			claudePart := t.convertUnifiedPartToClaude(part)
			if claudePart != nil {
				resp.Content = append(resp.Content, *claudePart)
			}
		}

		// Handle tool calls
		for _, toolCall := range choice.Message.ToolCalls {
			resp.Content = append(resp.Content, claude.ClaudeMediaMessage{
				Type:  "tool_use",
				Id:    toolCall.ID,
				Name:  toolCall.Name,
				Input: toolCall.Arguments,
			})
		}
	}

	return resp, nil
}

// Helper functions

func (t *ClaudeTransformer) extractTextFromMediaMessages(parts []claude.ClaudeMediaMessage) string {
	var text string
	for _, part := range parts {
		if part.Type == "text" {
			text += part.GetText()
		}
	}
	return text
}

func (t *ClaudeTransformer) convertClaudePartToUnified(part claude.ClaudeMediaMessage) *UnifiedMessagePart {
	switch part.Type {
	case "image":
		if part.Source != nil {
			unifiedPart := &UnifiedMessagePart{
				Type:      "image",
				MediaType: part.Source.MediaType,
			}

			switch part.Source.Type {
			case "base64":
				if data, ok := part.Source.Data.(string); ok {
					unifiedPart.Data = data
				}
			case "url":
				unifiedPart.ImageURL = &UnifiedImageURL{
					URL: part.Source.Url,
				}
			}

			return unifiedPart
		}
	case "document":
		// Handle document type if needed
		return &UnifiedMessagePart{
			Type: "document",
			Metadata: map[string]interface{}{
				"source": part.Source,
			},
		}
	}
	return nil
}

func (t *ClaudeTransformer) convertUnifiedPartToClaude(part UnifiedMessagePart) *claude.ClaudeMediaMessage {
	switch part.Type {
	case "image":
		claudePart := &claude.ClaudeMediaMessage{
			Type: "image",
		}

		if part.Data != "" {
			claudePart.Source = &claude.ClaudeMessageSource{
				Type:      "base64",
				MediaType: part.MediaType,
				Data:      part.Data,
			}
		} else if part.ImageURL != nil {
			claudePart.Source = &claude.ClaudeMessageSource{
				Type: "url",
				Url:  part.ImageURL.URL,
			}
		}

		return claudePart

	case "document":
		return &claude.ClaudeMediaMessage{
			Type: "document",
			Source: &claude.ClaudeMessageSource{
				Type:      "base64",
				MediaType: part.MediaType,
				Data:      part.Data,
			},
		}
	}
	return nil
}
