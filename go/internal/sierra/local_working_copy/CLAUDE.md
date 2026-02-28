# local_working_copy

Main repository interface coordinating store, workspace, and index operations for local working copies.

## Key Types

- `Repo`: Central repository coordinator with store, indexes, and workspace environments
- `Options`: Configuration for repository initialization

## Features

- Initializes and coordinates all major subsystems (store, workspace, indexes)
- Manages dormant index and ID abbreviation index
- Handles Lua environment for scripting
- Provides workspace store management for different workspace types
- Supports pull, checkin, reindex, and organize operations
