package gemini

import (
	"encoding/json"
)

type GeminiChatRequest struct {
	Contents           []GeminiChatContent        `json:"contents"`
	SafetySettings     []GeminiChatSafetySettings `json:"safetySettings,omitempty"`
	GenerationConfig   GeminiChatGenerationConfig `json:"generationConfig,omitempty"`
	Tools              []GeminiChatTool           `json:"tools,omitempty"`
	SystemInstructions *GeminiChatContent         `json:"systemInstruction,omitempty"`
}

type GeminiChatGenerationConfig struct {
	Temperature        *float64              `json:"temperature,omitempty"`
	TopP               float64               `json:"topP,omitempty"`
	TopK               float64               `json:"topK,omitempty"`
	MaxOutputTokens    uint                  `json:"maxOutputTokens,omitempty"`
	CandidateCount     int                   `json:"candidateCount,omitempty"`
	StopSequences      []string              `json:"stopSequences,omitempty"`
	ResponseMimeType   string                `json:"responseMimeType,omitempty"`
	ResponseSchema     any                   `json:"responseSchema,omitempty"`
	Seed               int64                 `json:"seed,omitempty"`
	ResponseModalities []string              `json:"responseModalities,omitempty"`
	ThinkingConfig     *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
	SpeechConfig       json.RawMessage       `json:"speechConfig,omitempty"`
}

type GeminiThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"`
	ThinkingBudget  *int `json:"thinkingBudget,omitempty"`
}

type GeminiChatTool struct {
	GoogleSearch          any `json:"googleSearch,omitempty"`
	GoogleSearchRetrieval any `json:"googleSearchRetrieval,omitempty"`
	CodeExecution         any `json:"codeExecution,omitempty"`
	FunctionDeclarations  any `json:"functionDeclarations,omitempty"`
}

type GeminiChatContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text                string                         `json:"text,omitempty"`
	Thought             bool                           `json:"thought,omitempty"`
	InlineData          *GeminiInlineData              `json:"inlineData,omitempty"`
	FunctionCall        *FunctionCall                  `json:"functionCall,omitempty"`
	FunctionResponse    *FunctionResponse              `json:"functionResponse,omitempty"`
	FileData            *GeminiFileData                `json:"fileData,omitempty"`
	ExecutableCode      *GeminiPartExecutableCode      `json:"executableCode,omitempty"`
	CodeExecutionResult *GeminiPartCodeExecutionResult `json:"codeExecutionResult,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type FunctionCall struct {
	FunctionName string `json:"name"`
	Arguments    any    `json:"args"`
}

type FunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiFileData struct {
	MimeType string `json:"mimeType,omitempty"`
	FileUri  string `json:"fileUri,omitempty"`
}

type GeminiPartExecutableCode struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code,omitempty"`
}

type GeminiPartCodeExecutionResult struct {
	Outcome string `json:"outcome,omitempty"`
	Output  string `json:"output,omitempty"`
}

type GeminiChatSafetySettings struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiChatCandidate struct {
	Content           GeminiChatContent        `json:"content"`
	FinishReason      *string                  `json:"finishReason"`
	Index             int64                    `json:"index"`
	SafetyRatings     []GeminiChatSafetyRating `json:"safetyRatings"`
	GroundingMetadata json.RawMessage          `json:"groundingMetadata,omitempty"`
}

type GeminiChatSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type GeminiChatPromptFeedback struct {
	SafetyRatings []GeminiChatSafetyRating `json:"safetyRatings"`
}

type GeminiChatResponse struct {
	Candidates     []GeminiChatCandidate    `json:"candidates"`
	PromptFeedback GeminiChatPromptFeedback `json:"promptFeedback"`
	UsageMetadata  GeminiUsageMetadata      `json:"usageMetadata"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount        int                         `json:"promptTokenCount"`
	CandidatesTokenCount    int                         `json:"candidatesTokenCount"`
	TotalTokenCount         int                         `json:"totalTokenCount"`
	ThoughtsTokenCount      int                         `json:"thoughtsTokenCount"`
	CachedContentTokenCount int                         `json:"cachedContentTokenCount"`
	PromptTokensDetails     []GeminiPromptTokensDetails `json:"promptTokensDetails"`

	Raw json.RawMessage `json:"-"`
}

func (m *GeminiUsageMetadata) UnmarshalJSON(data []byte) error {
	type Alias GeminiUsageMetadata
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*m = GeminiUsageMetadata(*aux)
	m.Raw = data
	return nil
}

type GeminiPromptTokensDetails struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}

type GeminiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
