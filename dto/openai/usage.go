package openai

// Usage represents token usage information for OpenAI API responses
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`  
	TotalTokens      int `json:"total_tokens"`
}