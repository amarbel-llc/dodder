# env_local

Composite environment interface combining UI and directory environments.

## Key Types

- `Env`: Interface embedding both `env_ui.Env` and `env_dir.Env`
- `env`: Implementation struct composing UI and directory environments

## Key Functions

- `Make`: Constructs composite environment from UI and directory components

## Purpose

Provides unified environment context for operations requiring both UI output capabilities and directory/filesystem context. Simplifies passing environment dependencies by combining two commonly-used environment types.
