# remote_http

HTTP-based remote synchronization client and server.

## Client Components

- `client`: HTTP client for remote operations
- `RoundTripper*`: Transport implementations (stdio, unix socket, retry)

## Server Components

- `server`: HTTP server for remote access
- `ServerRepo`: Repository operations handler
- `ServerBlobCache`: Blob caching layer
- `ServerMCP`: MCP protocol support
