package memory

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

// Manager provides unified interface for multi-level memory management
type Manager struct {
	config     *Config
	shortTerm  *ShortTermMemory
	longTerm   *LongTermMemory
	extractor  *Extractor
	aiProvider ai.AIProvider
}

// NewManager creates a new memory manager
func NewManager(config *Config, aiProvider ai.AIProvider) (*Manager, error) {
	if !config.Enabled {
		return nil, nil // Memory disabled
	}

	storagePath := expandPath(config.StoragePath)

	return &Manager{
		config:     config,
		shortTerm:  NewShortTermMemory(storagePath, config.ShortTermMaxTokens),
		longTerm:   NewLongTermMemory(storagePath, config.EntityThreshold),
		extractor:  NewExtractor(aiProvider),
		aiProvider: aiProvider,
	}, nil
}

// OnSessionEnd processes a completed conversation
func (m *Manager) OnSessionEnd(conv *conversation.Conversation) error {
	if m == nil {
		return nil
	}

	go m.processSessionEndAsync(conv)
	return nil
}

// processSessionEndAsync handles the async workflow
func (m *Manager) processSessionEndAsync(conv *conversation.Conversation) {
	ctx := context.Background()

	// Step 1: Generate summary
	summary, err := m.generateSummary(ctx, conv)
	if err != nil {
		return // Silently fail on error
	}

	// Step 2: Write to short-term memory
	summaryRecord := &Summary{
		ConversationID: conv.ID,
		Summary:        summary,
		Timestamp:      conv.UpdatedAt,
		Tokens:         estimateTokens(summary),
	}
	m.shortTerm.AddSummary(summaryRecord)

	// Step 3: Extract entities using LLM
	extraction, err := m.extractor.ExtractFromSummary(ctx, summary)
	if err != nil {
		return // Fallback: no extraction
	}

	// Step 4: Update entity counts and check for promotion
	for _, entity := range extraction.Entities {
		promoted, _ := m.longTerm.UpdateEntity(entity)
		if promoted {
			// Entity promoted to profile
		}
	}

	// Step 5: Update profile from extraction results
	m.longTerm.UpdateProfile(extraction)
}

// generateSummary creates a summary of the conversation
func (m *Manager) generateSummary(ctx context.Context, conv *conversation.Conversation) (string, error) {
	// Build messages from conversation
	messages := []ai.Message{
		{Role: "system", Content: "Summarize the following conversation in 1-2 sentences, focusing on key topics discussed."},
	}

	for _, msg := range conv.Messages {
		messages = append(messages, ai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	summary, err := m.aiProvider.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(summary), nil
}

// BuildContext constructs messages with memory context for AI calls
func (m *Manager) BuildContext(currentMessages []ai.Message) []ai.Message {
	if m == nil {
		return currentMessages
	}

	systemPrompt := m.buildSystemPrompt()

	// Prepend system prompt with memory context
	result := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	return append(result, currentMessages...)
}

// buildSystemPrompt creates system prompt with memory context
func (m *Manager) buildSystemPrompt() string {
	var parts []string

	// User profile (L3)
	profile := m.longTerm.GetProfile()
	if len(profile.TechPreferences.Languages) > 0 || len(profile.TechPreferences.Frameworks) > 0 {
		parts = append(parts, "## User Profile")
		if len(profile.TechPreferences.Languages) > 0 {
			parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(profile.TechPreferences.Languages, ", ")))
		}
		if len(profile.TechPreferences.Frameworks) > 0 {
			parts = append(parts, fmt.Sprintf("Frameworks: %s", strings.Join(profile.TechPreferences.Frameworks, ", ")))
		}
	}

	// Short-term memory summaries (L2)
	summaries := m.shortTerm.GetSummaries()
	if len(summaries) > 0 {
		parts = append(parts, "## Recent Conversations")
		for _, s := range summaries {
			parts = append(parts, fmt.Sprintf("- %s", s.Summary))
		}
	}

	if len(parts) == 0 {
		return "You are tada, a terminal AI assistant."
	}

	return fmt.Sprintf(`You are tada, a terminal AI assistant.

%s

Use this context to provide more personalized responses.`, strings.Join(parts, "\n"))
}

// estimateTokens roughly estimates token count (1 token ≈ 4 characters)
func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "/root" // Fallback
		}
		return strings.Replace(path, "~", homeDir, 1)
	}
	return path
}
