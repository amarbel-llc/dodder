# mcp

Model Context Protocol (MCP) types for server integration.

## Core Types

- `Request`, `Response`: JSON-RPC 2.0 message structures
- `Error`: Structured error response with code, message, and data
- `Resource`: Resource descriptor with URI, name, description, and MIME type

## Resource Operations

- `ResourcesListResult`: List of available resources
- `ResourcesReadParams`, `ResourcesReadResult`: Resource reading operations
- `ResourceContent`: Resource content with text or blob data

## Server Metadata

- `InitializeResponse`: Server initialization with protocol version and capabilities
- `ServerCapabilities`: Feature flags for resources and tools
- `ServerInfo`: Server name and version

Defines the MCP protocol for exposing Dodder data to LLM integrations.
