# commands_madder

Low-level "madder" CLI commands for blob and repository operations.

## Key Commands

- `cat`: Output blob contents by SHA, optionally with external utility processing
- `cat_ids`: Output object IDs
- `complete`: Shell completion support
- `fsck`: Filesystem consistency check
- `info_repo`: Repository information display

## Features

- Blob store operations with prefix SHA output option
- External utility piping for blob processing
- Uses command framework from kilo/command
