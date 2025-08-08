package transformer

import (
	"context"
)

// Provider represents supported LLM providers
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderGemini Provider = "gemini"
	ProviderClaude Provider = "claude"
)

// UnifiedRequest represents a normalized request format that all providers can transform to/from
type UnifiedRequest struct {
	Model            string                 `json:"model"`
	Messages         []UnifiedMessage       `json:"messages"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Tools            []UnifiedTool          `json:"tools,omitempty"`
	ToolChoice       string                 `json:"tool_choice,omitempty"`
	StopSequences    []string               `json:"stop_sequences,omitempty"`
	ResponseFormat   *UnifiedResponseFormat `json:"response_format,omitempty"`
	Seed             *int64                 `json:"seed,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
}

// UnifiedMessage represents a normalized message format
type UnifiedMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	Name       string               `json:"name,omitempty"`
	Parts      []UnifiedMessagePart `json:"parts,omitempty"`
	ToolCalls  []UnifiedToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

// UnifiedMessagePart represents content parts (text, image, etc.)
type UnifiedMessagePart struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ImageURL  *UnifiedImageURL       `json:"image_url,omitempty"`
	MediaType string                 `json:"media_type,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UnifiedImageURL represents image content
type UnifiedImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// UnifiedTool represents function/tool definitions
type UnifiedTool struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// UnifiedToolCall represents tool invocations
type UnifiedToolCall struct {
	ID        string                 `json:"id,omitempty"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// UnifiedResponseFormat represents structured output format
type UnifiedResponseFormat struct {
	Type   string                 `json:"type"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// UnifiedResponse represents a normalized response format
type UnifiedResponse struct {
	ID                string                 `json:"id,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Created           int64                  `json:"created,omitempty"`
	Model             string                 `json:"model,omitempty"`
	Provider          Provider               `json:"provider"`
	Choices           []UnifiedChoice        `json:"choices"`
	Usage             *UnifiedUsage          `json:"usage,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Error             *UnifiedError          `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// UnifiedChoice represents response choices
type UnifiedChoice struct {
	Index        int              `json:"index"`
	Message      UnifiedMessage   `json:"message"`
	FinishReason string           `json:"finish_reason,omitempty"`
	LogProbs     *UnifiedLogProbs `json:"logprobs,omitempty"`
}

// UnifiedUsage represents token usage information
type UnifiedUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
}

// UnifiedLogProbs represents log probability information
type UnifiedLogProbs struct {
	Content []UnifiedLogProb `json:"content,omitempty"`
}

// UnifiedLogProb represents individual token probabilities
type UnifiedLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// UnifiedError represents error information
type UnifiedError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Error implements the error interface
func (e *UnifiedError) Error() string {
	return e.Message
}

// Transformer interface defines methods for converting between provider formats and unified format
type Transformer interface {
	// ToUnified converts provider-specific request to unified format
	ToUnified(ctx context.Context, providerRequest interface{}) (*UnifiedRequest, error)

	// FromUnified converts unified request to provider-specific format
	FromUnified(ctx context.Context, unifiedRequest *UnifiedRequest) (interface{}, error)

	// ResponseToUnified converts provider-specific response to unified format
	ResponseToUnified(ctx context.Context, providerResponse interface{}) (*UnifiedResponse, error)

	// ResponseFromUnified converts unified response to provider-specific format
	ResponseFromUnified(ctx context.Context, unifiedResponse *UnifiedResponse) (interface{}, error)

	// GetProvider returns the provider this transformer handles
	GetProvider() Provider

	// ValidateRequest validates the provider-specific request
	ValidateRequest(ctx context.Context, request interface{}) error
}

// TransformationPair represents a source->target transformation
type TransformationPair struct {
	Source Provider
	Target Provider
}

// TransformationRegistry manages all available transformers
type TransformationRegistry struct {
	transformers map[Provider]Transformer
}

// NewTransformationRegistry creates a new transformation registry
func NewTransformationRegistry() *TransformationRegistry {
	return &TransformationRegistry{
		transformers: make(map[Provider]Transformer),
	}
}

// Register adds a transformer to the registry
func (r *TransformationRegistry) Register(transformer Transformer) {
	r.transformers[transformer.GetProvider()] = transformer
}

// Transform converts a request from source provider format to target provider format
func (r *TransformationRegistry) Transform(ctx context.Context, sourceProvider, targetProvider Provider, request interface{}) (interface{}, error) {
	// Get source transformer
	sourceTransformer, exists := r.transformers[sourceProvider]
	if !exists {
		return nil, &UnifiedError{
			Type:    "transformer_not_found",
			Message: "transformer not found for source provider: " + string(sourceProvider),
		}
	}

	// Get target transformer
	targetTransformer, exists := r.transformers[targetProvider]
	if !exists {
		return nil, &UnifiedError{
			Type:    "transformer_not_found",
			Message: "transformer not found for target provider: " + string(targetProvider),
		}
	}

	// Convert source to unified format
	unified, err := sourceTransformer.ToUnified(ctx, request)
	if err != nil {
		return nil, err
	}

	// Convert unified to target format
	return targetTransformer.FromUnified(ctx, unified)
}

// TransformResponse converts a response from source provider format to target provider format
func (r *TransformationRegistry) TransformResponse(ctx context.Context, sourceProvider, targetProvider Provider, response interface{}) (interface{}, error) {
	// Get source transformer
	sourceTransformer, exists := r.transformers[sourceProvider]
	if !exists {
		return nil, &UnifiedError{
			Type:    "transformer_not_found",
			Message: "transformer not found for source provider: " + string(sourceProvider),
		}
	}

	// Get target transformer
	targetTransformer, exists := r.transformers[targetProvider]
	if !exists {
		return nil, &UnifiedError{
			Type:    "transformer_not_found",
			Message: "transformer not found for target provider: " + string(targetProvider),
		}
	}

	// Convert source response to unified format
	unified, err := sourceTransformer.ResponseToUnified(ctx, response)
	if err != nil {
		return nil, err
	}

	// Convert unified to target response format
	return targetTransformer.ResponseFromUnified(ctx, unified)
}

// GetAvailableTransformations returns all possible transformation pairs
func (r *TransformationRegistry) GetAvailableTransformations() []TransformationPair {
	var pairs []TransformationPair
	providers := make([]Provider, 0, len(r.transformers))

	for provider := range r.transformers {
		providers = append(providers, provider)
	}

	for _, source := range providers {
		for _, target := range providers {
			if source != target {
				pairs = append(pairs, TransformationPair{
					Source: source,
					Target: target,
				})
			}
		}
	}

	return pairs
}

// GetSupportedProviders returns all registered providers
func (r *TransformationRegistry) GetSupportedProviders() []Provider {
	providers := make([]Provider, 0, len(r.transformers))
	for provider := range r.transformers {
		providers = append(providers, provider)
	}
	return providers
}
