package memory

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// Conversation represents a conversation for memory processing
type Conversation interface {
	ID() string
	GetMessages() []ConversationMessage
	UpdatedAt() time.Time
}

// ConversationMessage represents a message in a conversation
type ConversationMessage interface {
	Role() string
	Content() string
}

// Manager provides unified interface for multi-level memory management
type Manager struct {
	config       *Config
	shortTerm    *ShortTermMemory
	longTerm     *LongTermMemory
	extractor    *Extractor
	aiProvider   ai.AIProvider
	promptLoader *PromptLoader
}

// NewManager creates a new memory manager
func NewManager(config *Config, aiProvider ai.AIProvider, promptLoader *PromptLoader) (*Manager, error) {
	if !config.Enabled {
		return nil, nil // Memory disabled
	}

	storagePath := expandPath(config.StoragePath)

	longTerm := NewLongTermMemory(storagePath, config.EntityThreshold)
	longTerm.SetAIProvider(aiProvider, promptLoader)

	return &Manager{
		config:       config,
		shortTerm:    NewShortTermMemory(storagePath, config.ShortTermMaxTokens),
		longTerm:     longTerm,
		extractor:    NewExtractor(aiProvider, promptLoader),
		aiProvider:   aiProvider,
		promptLoader: promptLoader,
	}, nil
}

// OnSessionEnd processes a completed conversation
func (m *Manager) OnSessionEnd(conv Conversation) error {
	if m == nil {
		return nil
	}

	// Start async processing without blocking
	go m.processSessionEndAsync(conv)
	return nil
}

// processSessionEndAsync handles the async workflow
func (m *Manager) processSessionEndAsync(conv Conversation) {
	log.Printf("[memory] Processing session end for conversation %s", conv.ID())

	ctx := context.Background()

	// Step 1: Generate summary
	log.Printf("[memory] Step 1: Generating summary...")
	summary, err := m.generateSummary(ctx, conv)
	if err != nil {
		log.Printf("[memory] Error generating summary: %v", err)
		return
	}
	log.Printf("[memory] Summary generated: %s", summary)

	// Step 2: Write to short-term memory
	log.Printf("[memory] Step 2: Writing to short-term memory...")
	summaryRecord := &Summary{
		ConversationID: conv.ID(),
		Summary:        summary,
		Timestamp:      conv.UpdatedAt(),
		Tokens:         estimateTokens(summary),
	}
	m.shortTerm.AddSummary(summaryRecord)

	// Step 3: Extract entities using LLM
	log.Printf("[memory] Step 3: Extracting entities...")
	extraction, err := m.extractor.ExtractFromSummary(ctx, summary)
	if err != nil {
		log.Printf("[memory] Error extracting entities: %v", err)
		return // Fallback: no extraction
	}
	log.Printf("[memory] Entities extracted: %v", extraction.Entities)

	// Step 4: Update entity counts and check for promotion
	log.Printf("[memory] Step 4: Updating entity counts...")
	for _, entity := range extraction.Entities {
		promoted, _ := m.longTerm.UpdateEntity(entity)
		if promoted {
			log.Printf("[memory] Entity '%s' promoted to profile", entity)
		}
	}

	// Step 5: Update profile using LLM with new information
	log.Printf("[memory] Step 5: Updating user profile...")
	// Combine summary and extraction for comprehensive profile update
	newInfo := fmt.Sprintf("Conversation Summary: %s\n\nExtracted Entities: %s",
		summary, strings.Join(extraction.Entities, ", "))
	if len(extraction.Context) > 0 {
		newInfo += fmt.Sprintf("\nTopics Discussed: %s", strings.Join(extraction.Context, ", "))
	}
	err = m.longTerm.UpdateProfileWithLLM(ctx, newInfo)
	if err != nil {
		log.Printf("[memory] Error updating profile: %v", err)
		return
	}
	log.Printf("[memory] Session processing complete!")
}

// generateSummary creates a summary of the conversation
func (m *Manager) generateSummary(ctx context.Context, conv Conversation) (string, error) {
	// Get summary prompt from loader or use default
	summaryPrompt := "Summarize the following conversation in 1-2 sentences, focusing on key topics discussed."
	if m.promptLoader != nil {
		summaryPrompt = m.promptLoader.LoadOrDefault("summary", summaryPrompt)
	}

	// Build messages from conversation
	messages := []ai.Message{
		{Role: "system", Content: summaryPrompt},
	}

	for _, msg := range conv.GetMessages() {
		messages = append(messages, ai.Message{
			Role:    msg.Role(),
			Content: msg.Content(),
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
	// Get profile markdown directly
	profileMD := m.longTerm.GetProfileMarkdown()

	// Build summaries section (only the list items, template includes the header)
	summaries := m.shortTerm.GetSummaries()
	var summaryParts []string
	for _, s := range summaries {
		summaryParts = append(summaryParts, fmt.Sprintf("- %s", s.Summary))
	}

	// Try to load system template, or use default
	systemPrompt := "You are tada, a terminal AI assistant."
	if m.promptLoader != nil {
		template, err := m.promptLoader.Load("system")
		if err == nil {
			// Replace placeholders
			systemPrompt = template.SystemPrompt
			systemPrompt = strings.ReplaceAll(systemPrompt, "{{profile}}", profileMD)
			systemPrompt = strings.ReplaceAll(systemPrompt, "{{summaries}}", strings.Join(summaryParts, "\n"))
			return systemPrompt
		}
	}

	// Fallback to default behavior - add headers manually
	var parts []string
	if profileMD != "" {
		parts = append(parts, profileMD)
	}
	if len(summaryParts) > 0 {
		parts = append(parts, "## Recent Conversations")
		parts = append(parts, summaryParts...)
	}

	if len(parts) == 0 {
		return systemPrompt
	}

	return fmt.Sprintf(`%s

%s

Use this context to provide more personalized responses.`, systemPrompt, strings.Join(parts, "\n"))
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
