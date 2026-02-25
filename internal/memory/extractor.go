package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// Extractor uses LLM to extract entities and preferences from summaries
type Extractor struct {
	aiProvider ai.AIProvider
}

// NewExtractor creates a new LLM-based extractor
func NewExtractor(provider ai.AIProvider) *Extractor {
	return &Extractor{
		aiProvider: provider,
	}
}

// ExtractFromSummary extracts structured information from a conversation summary
func (e *Extractor) ExtractFromSummary(ctx context.Context, summary string) (*ExtractionResult, error) {
	prompt := e.buildExtractionPrompt(summary)

	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that extracts structured information from conversations. Always respond with valid JSON only."},
		{Role: "user", Content: prompt},
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
	return fmt.Sprintf(`Extract the following information from this conversation summary:

Summary: %s

Please extract and return as JSON:
{
  "entities": ["list of technologies, frameworks, tools mentioned"],
  "preferences": {"editor": "preferred editor if mentioned", "timezone": "timezone if mentioned", "shell": "shell if mentioned"},
  "context": ["key topics, projects, or areas of interest discussed"]
}

Only include fields that have values. Return valid JSON only, no markdown.`, summary)
}
