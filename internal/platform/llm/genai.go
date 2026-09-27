package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

var (
	ErrMissingAPIKey = errors.New("GEMINI_API_KEY is not configured")
	ErrEmptyResponse = errors.New("gemini returned an empty response")
)

var responseSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"reason":             {Type: genai.TypeString},
		"recommended_action": {Type: genai.TypeString},
		"confidence":         {Type: genai.TypeNumber},
		"insights":           {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
	},
	Required:         []string{"reason", "recommended_action", "confidence", "insights"},
	PropertyOrdering: []string{"reason", "recommended_action", "confidence", "insights"},
}

type GenaiGenerator struct {
	client *genai.Client
	model  string
}

func NewGenaiGenerator(ctx context.Context, apiKey, model string) (*GenaiGenerator, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrMissingAPIKey
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &GenaiGenerator{client: client, model: model}, nil
}

func (g *GenaiGenerator) Generate(ctx context.Context, system, prompt string) (string, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(system, genai.RoleUser),
		Temperature:       genai.Ptr[float32](0.2),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    responseSchema,
	})
	if err != nil {
		return "", err
	}
	if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("%w: blocked (%s)", ErrEmptyResponse, resp.PromptFeedback.BlockReason)
	}
	text := resp.Text()
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyResponse
	}
	return text, nil
}
