# webdav

WebDAV HTTP client for file-level operations (PROPFIND, GET, PUT, DELETE, MKCOL).

## Key Types

- `Client`: WebDAV HTTP client with basic auth
- `Config`: Connection parameters (URL, username, password)
- `Resource`: File/collection metadata from PROPFIND (href, content type, etag, size)

## Key Functions

- `MakeClient`: Construct a client from config
- `List`: PROPFIND to enumerate collection members
- `Get`: Retrieve file content
- `Put`: Create or update a file (with optional ETag conditional)
- `Delete`: Remove a resource
- `Mkcol`: Create a collection (directory)
