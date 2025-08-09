package transformer

import (
	"context"
	"strings"
)

// Provider represents supported LLM providers
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderGemini Provider = "gemini"
	ProviderClaude Provider = "claude"
)

type TransformerType string

const (
	TransformerTypeRequest  TransformerType = "request"
	TransformerTypeResponse TransformerType = "response"
	TransformerTypeStream   TransformerType = "stream"
	TransformerTypeChunk    TransformerType = "chunk"
)

// Transformer interface for one-to-one direct transformations with stream/chunk support
type Transformer interface {
	Do(ctx context.Context, typ TransformerType, src interface{}, dst interface{}) error

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

// TransformationRegistry manages all available transformers for direct one-to-one transformations
type TransformationRegistry struct {
	transformers map[string]Transformer // key format: "sourceProvider->targetProvider"
}

// NewTransformationRegistry creates a new transformation registry
func NewTransformationRegistry() *TransformationRegistry {
	return &TransformationRegistry{
		transformers: make(map[string]Transformer),
	}
}

// Register adds a transformer to the registry for a specific source->target pair
func (r *TransformationRegistry) Register(sourceProvider, targetProvider Provider, transformer Transformer) {
	key := string(sourceProvider) + "->" + string(targetProvider)
	r.transformers[key] = transformer
}

// GetTransformer returns the transformer for a specific source->target pair
func (r *TransformationRegistry) GetTransformer(sourceProvider, targetProvider Provider) (Transformer, bool) {
	key := string(sourceProvider) + "->" + string(targetProvider)
	transformer, exists := r.transformers[key]
	return transformer, exists
}

// Transform performs direct transformation from source to target format
func (r *TransformationRegistry) Transform(ctx context.Context, sourceProvider, targetProvider Provider, typ TransformerType, src interface{}, dst interface{}) error {
	transformer, exists := r.GetTransformer(sourceProvider, targetProvider)
	if !exists {
		return &TransformationError{
			Type:    "transformer_not_found",
			Message: "transformer not found for " + string(sourceProvider) + " -> " + string(targetProvider),
		}
	}

	return transformer.Do(ctx, typ, src, dst)
}

// GetAvailableTransformations returns all available transformation pairs
func (r *TransformationRegistry) GetAvailableTransformations() []TransformationPair {
	var pairs []TransformationPair
	
	for key := range r.transformers {
		parts := strings.Split(key, "->")
		if len(parts) == 2 {
			pairs = append(pairs, TransformationPair{
				Source: Provider(parts[0]),
				Target: Provider(parts[1]),
			})
		}
	}
	
	return pairs
}

// GetSupportedProviders returns all unique providers that have transformers
func (r *TransformationRegistry) GetSupportedProviders() []Provider {
	providerMap := make(map[Provider]bool)
	
	for key := range r.transformers {
		parts := strings.Split(key, "->")
		if len(parts) == 2 {
			providerMap[Provider(parts[0])] = true
			providerMap[Provider(parts[1])] = true
		}
	}
	
	var providers []Provider
	for provider := range providerMap {
		providers = append(providers, provider)
	}
	
	return providers
}

// TransformationError represents error information
type TransformationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Error implements the error interface
func (e *TransformationError) Error() string {
	return e.Message
}