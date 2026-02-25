package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// Extractor uses LLM to extract entities and preferences from summaries
type Extractor struct {
	aiProvider   ai.AIProvider
	promptLoader *PromptLoader
}

// NewExtractor creates a new LLM-based extractor
func NewExtractor(provider ai.AIProvider, promptLoader *PromptLoader) *Extractor {
	return &Extractor{
		aiProvider:   provider,
		promptLoader: promptLoader,
	}
}

// ExtractFromSummary extracts structured information from a conversation summary
func (e *Extractor) ExtractFromSummary(ctx context.Context, summary string) (*ExtractionResult, error) {
	// Get extraction prompt
	extractPrompt := e.buildExtractionPrompt(summary)

	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that extracts structured information from conversations. Always respond with valid JSON only."},
		{Role: "user", Content: extractPrompt},
	}

	response, err := e.aiProvider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse JSON response
	var result ExtractionResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: return empty result if parsing fails
		return &ExtractionResult{}, nil
	}

	return &result, nil
}

// buildExtractionPrompt creates the prompt for entity extraction
func (e *Extractor) buildExtractionPrompt(summary string) string {
	// Try to load extract template, or use default
	defaultPrompt := `Extract the following information from this conversation summary:

Summary: {{summary}}

Please extract and return as JSON:
{
  "entities": ["list of technologies, frameworks, tools mentioned"],
  "preferences": {"editor": "preferred editor if mentioned", "timezone": "timezone if mentioned", "shell": "shell if mentioned"},
  "context": ["key topics, projects, or areas of interest discussed"]
}

Only include fields that have values. Return valid JSON only, no markdown.`

	var extractPrompt string
	if e.promptLoader != nil {
		template, err := e.promptLoader.Load("extract")
		if err == nil {
			extractPrompt = strings.ReplaceAll(template.SystemPrompt, "{{summary}}", summary)
			return extractPrompt
		}
	}

	// Fallback to default
	return fmt.Sprintf(strings.ReplaceAll(defaultPrompt, "{{summary}}", "%s"), summary)
}
