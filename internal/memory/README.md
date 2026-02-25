# Memory Package

Multi-level memory system for tada chat functionality.

## Architecture

- **L1**: Current session (handled by conversation package)
- **L2**: Short-term memory (summaries.json) - recent conversation summaries
- **L3**: Long-term memory (user_profile.json, entities.json) - persistent knowledge

## Components

- `types.go`: Core data structures
- `short_term.go`: Short-term memory manager
- `long_term.go`: Long-term memory manager
- `profiler.go`: User profile manager
- `extractor.go`: LLM-based entity extractor
- `manager.go`: Unified management interface
