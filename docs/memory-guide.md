# Memory System Guide

## How It Works

The memory system learns from your conversations to provide more personalized assistance. It uses a multi-level architecture inspired by CPU caches for efficient context management.

## Levels

### L1: Current Session
Your ongoing conversation with full message history. This is what you see during a chat session.

### L2: Short-term Memory
Recent conversation summaries, automatically generated when sessions end.
- Limited by token count (default: 4000)
- Oldest summaries removed when full (FIFO eviction)
- Stored in: `~/.tada/memory/summaries.json`

### L3: Long-term Memory
**User Profile**: Your learned preferences (languages, frameworks, editor, etc.)
**Entity Tracking**: Topics mentioned repeatedly across sessions
- Stored in: `~/.tada/memory/user_profile.json` and `entities.json`

## Entity Promotion

When something is mentioned in 5+ different sessions, it's automatically promoted to your user profile. This helps the AI remember your long-term preferences.

**Example:** If you mention "Go" programming in 5 different chat sessions, it will be added to your profile under "Languages", and the AI will remember you prefer Go for future tasks.

## Configuration

Edit `~/.tada/config.yaml` to customize memory behavior:

```yaml
memory:
  enabled: true                      # Enable/disable memory system
  short_term_max_tokens: 4000        # L2 token limit
  entity_threshold: 5                # Mentions needed for promotion
  storage_path: "~/.tada/memory"     # Storage location
```

### Settings Explained

- **enabled**: Turn memory on/off. When disabled, no memory is stored or used.
- **short_term_max_tokens**: How much conversation history to keep in summaries. Higher = more context, but slower responses.
- **entity_threshold**: How many times something must be mentioned before being added to your profile.
- **storage_path**: Where memory files are stored. Defaults to `~/.tada/memory`.

## Privacy

- **Local storage**: All memory stored locally in `~/.tada/memory/`
- **No external transmission**: Data only sent to LLM for processing (summary generation, entity extraction)
- **Can be disabled**: Set `memory.enabled: false` in config
- **Manual control**: You can edit or delete memory files at any time
- **Ephemeral mode**: Use `--no-history` flag to disable memory for a session

## Viewing Your Memory

```bash
# View current profile
cat ~/.tada/memory/user_profile.json

# View tracked entities
cat ~/.tada/memory/entities.json

# View recent summaries
cat ~/.tada/memory/summaries.json
```

## Example Profile

```json
{
  "tech_preferences": {
    "languages": ["Go", "Python"],
    "frameworks": ["React", "Gin"],
    "editors": ["neovim", "vscode"]
  },
  "work_context": {
    "current_projects": ["tada", "side-project"],
    "common_paths": ["/Users/johnlin/Dev/go/tada"]
  },
  "behavior_patterns": {
    "preferred_communication": "concise",
    "often_asks": ["code review", "debugging help"]
  },
  "personal_settings": {
    "timezone": "Asia/Shanghai",
    "shell": "zsh"
  }
}
```

## Tips

1. **Consistency helps**: Mentioning the same tools/languages across sessions helps build a better profile
2. **Ephemeral for sensitive**: Use `--no-history` for conversations you don't want remembered
3. **Manual editing**: You can manually add items to your profile JSON file
4. **Reset anytime**: Delete memory files to start fresh

## Troubleshooting

**Memory not working?**
- Check `~/.tada/config.yaml` has `memory.enabled: true`
- Verify API key is configured (memory uses LLM for summaries/extraction)

**Too much memory used?**
- Reduce `short_term_max_tokens` in config
- Delete old entries from `summaries.json`

**Want to forget something?**
- Edit the relevant JSON file in `~/.tada/memory/`
- Or delete all memory files to start fresh
