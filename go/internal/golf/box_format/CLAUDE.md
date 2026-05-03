# box_format

Box-style text formatting for checked-out and transacted objects.

## Purpose

Formats SKU objects as structured text boxes for terminal output with color support.

## Key Types

- `BoxCheckedOut`: Formatter for checked-out objects with filesystem paths
- `BoxTransacted`: Base formatter for transacted objects

## Features

- Displays metadata (ID, tags, type, description) in structured boxes
- Handles different checkout states (untracked, recognized, conflicted)
- Supports filesystem path abbreviation and relative paths
- Optional field display (blob digests, TAI timestamps)
- Color-coded output with customizable options
