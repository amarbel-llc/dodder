# alfred

Alfred workflow JSON output formatting for macOS Alfred app integration.

## Key Types

- `Item`: Alfred search result with title, subtitle, icon, and action modifiers
- `Mod`: Keyboard modifier actions (cmd, alt, etc.) for items
- `Writer`: Interface for writing items to output with debouncing support
- `ItemPool`: Thread-safe object pool for Item allocation

## Features

- JSON-formatted output for Alfred workflows
- Debounced writing to prevent UI flickering
- Pooled Item allocation for performance
- Support for icons, quicklook, and copy text
- Keyboard modifier actions
