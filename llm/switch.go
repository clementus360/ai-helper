package llm

import (
	"clementus360/ai-helper/types"
	"fmt"
)

type Model string

const (
	OpenAI Model = "openai"
	Gemini Model = "gemini"
)

// GenerateResponse generates a response using the specified AI model
func GenerateResponse(userInput string, context types.SmartContext, model Model) (GeminiStructuredResponse, error) {
	switch model {
	case OpenAI:
		return OpenAIGenerateResponse(userInput, context)
	case Gemini:
		return GeminiGenerateResponse(userInput, context)
	default:
		return GeminiStructuredResponse{}, fmt.Errorf("unsupported model: %s (supported: %s, %s)", model, OpenAI, Gemini)
	}
}
