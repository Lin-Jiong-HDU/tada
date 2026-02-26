# Multi-Level Memory System Design

**Date**: 2025-01-24
**Status**: Design Approved

## Overview

Implement a CPU cache-inspired multi-level memory system for tada's chat functionality, enabling long-term memory and user profiling.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                     L1 (Current Session)            │
│  internal/conversation - existing implementation     │
└─────────────────────────────────────────────────────┘
                        ↓ triggers on session end
┌─────────────────────────────────────────────────────┐
│                Short-term Memory (Summaries)         │
│  ~/.tada/memory/summaries.json                       │
│  - Recent conversation summaries                     │
│  - Token-based capacity management                   │
└─────────────────────────────────────────────────────┘
                        ↓ async processing
┌─────────────────────────────────────────────────────┐
│                 Long-term Memory                     │
│  ~/.tada/memory/user_profile.json  (User Profile)    │
│  ~/.tada/memory/entities.json      (Entity Counts)  │
└─────────────────────────────────────────────────────┘
```

## Data Structures

### `summaries.json` - Short-term Memory
```json
{
  "max_tokens": 4000,
  "summaries": [
    {
      "conversation_id": "conv-20250124-001",
      "summary": "User asked about Go memory management, discussed GC and escape analysis",
      "timestamp": "2025-01-24T10:30:00Z",
      "tokens": 45
    }
  ]
}
```

### `entities.json` - Entity Tracking
```json
{
  "entities": {
    "Go语言": {"count": 5, "first_seen": "2025-01-20", "last_seen": "2025-01-24"},
    "React": {"count": 3, "first_seen": "2025-01-22", "last_seen": "2025-01-23"}
  }
}
```

### `user_profile.json` - User Profile
```json
{
  "tech_preferences": {
    "languages": ["Go", "TypeScript"],
    "frameworks": ["React", "Gin"],
    "editors": ["neovim"]
  },
  "work_context": {
    "current_projects": ["tada"],
    "common_paths": ["/Users/johnlin/Dev/go"]
  },
  "behavior_patterns": {
    "preferred_communication": "简洁直接",
    "often_asks": ["架构设计", "性能优化"]
  },
  "personal_settings": {
    "timezone": "Asia/Shanghai",
    "shell": "zsh"
  }
}
```

## Components

### New Package: `internal/memory`

```
internal/memory/
├── types.go           # Data structure definitions
├── short_term.go      # Short-term memory manager
├── long_term.go       # Long-term memory manager
├── profiler.go        # User profile manager
├── extractor.go       # LLM-based entity extractor
└── manager.go         # Unified management interface
```

### Core Component Responsibilities

**Extractor** (LLM-based Entity Extraction):
- Extract entities, preferences, and context from summaries using LLM
- Returns structured extraction results

**ShortTermMemory**:
- Manage conversation summaries with token limit
- FIFO eviction when capacity exceeded
- Provide summaries for prompt construction

**LongTermMemory**:
- Track entity occurrence counts
- Promote entities to profile when threshold (5) reached
- Manage user profile updates

**Manager**:
- Unified interface for memory operations
- Coordinate L1→L2→L3 flow on session end
- Build complete context for AI calls

## Data Flow

### Session End Processing
```
Session End → Generate Summary → Write to L2 → LLM Extract Entities → Update Counts → Check Threshold → Update Profile
```

### Prompt Building
```
User Query → Load L1 → Load L2 → Load L3 → Construct Messages → Call AI
```

## Error Handling

| Scenario | Strategy |
|----------|----------|
| Summary generation fails | Log error, skip this summary |
| Entity extraction fails | Fallback to keyword extraction |
| File write fails | Retry 3 times, then log error |
| File read fails | Return empty memory, log warning |
| Token limit exceeded | Delete oldest summary (FIFO) |

## Configuration

Add to `~/.tada/config.yaml`:
```yaml
memory:
  enabled: true
  short_term:
    max_tokens: 4000
  long_term:
    entity_threshold: 5
  storage_path: "~/.tada/memory"
```

## Integration Points

1. **`internal/conversation/manager.go`**: Call memory on session end
2. **AI Chat Entry**: Inject memory context when building messages

## Testing Strategy

1. Unit tests for each component
2. Integration tests for complete flow
3. Mock AI provider for testing
