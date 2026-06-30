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

## Merge resolution (#299)

The two transfer directions resolve the merge base differently because of who
imports:

- **Pull** imports locally (`local_op_pull` + `ParentNegotiatorFirstAncestor`),
  fetching the remote's history out of band over `GET /object-history`.
- **Push** imports server-side: the client `POST`s the source's inventory list
  to `/inventory_lists` and the server's
  `writeInventoryListTypedBlobLocalWorkingCopy` imports it. The server cannot
  query the pushing client, so the client first expands the list to each
  object's full history (`local_working_copy.ExpandListToObjectHistory`) and the
  server builds a `local_working_copy.ParentNegotiatorInBand` from the POSTed
  list (decode-twice over the buffered body: once to populate the negotiator,
  once for the blob-digest-validating import). A genuine divergence then returns
  500 and fails the push, instead of silently overwriting the receiver's
  version. `ExpandListToObjectHistory` / `ParentNegotiatorInBand` are shared
  with the drtp transport (`sierra/remote_proto`).
