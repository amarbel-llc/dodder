package haustoria_orgmode

// Transport abstracts file operations for orgmode haustoria. Two
// implementations exist: WebDAV (HTTP) and SFTP (SSH).
type Transport interface {
	// List returns the orgmode files in the given folder URL/path.
	List(folder string) ([]RemoteFile, error)

	// Read retrieves the content and ETag of a file.
	Read(filePath string) (content []byte, etag string, err error)

	// Write creates or updates a file. If etag is non-empty, the transport
	// should attempt a conditional update.
	Write(filePath string, content []byte, etag string) error

	// Delete removes a file.
	Delete(filePath string) error

	// Close releases any resources held by the transport (connections, etc.).
	Close() error
}

// RemoteFile is a file discovered by Transport.List.
type RemoteFile struct {
	Path string
	Name string
	ETag string
	Size int64
}
