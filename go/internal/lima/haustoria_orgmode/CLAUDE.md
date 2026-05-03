# haustoria_orgmode

Orgmode haustoria implementation for syncing dodder zettels with orgmode files
over WebDAV or SFTP.

## Key Types

- `Store`: Implements both haustoria.Haustoria and store_workspace.StoreLike
- `Transport`: Interface abstracting file operations (list, read, write, delete)
- `webdavTransport`: Transport implementation backed by hotel/webdav
- `sftpTransport`: Transport implementation backed by SFTP
- `FolderMapping`: Maps a remote folder to a dodder type and tags

## Key Functions

- `MakeStore`: Construct a store with transport and folder mappings
- `MakeWebDAVTransport`: Create transport from WebDAV config
- `MakeSFTPTransport`: Create transport from SFTP config
