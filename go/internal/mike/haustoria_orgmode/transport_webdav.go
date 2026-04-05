package haustoria_orgmode

import (
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/hotel/webdav"
)

// webdavTransport implements Transport using a WebDAV client.
type webdavTransport struct {
	client *webdav.Client
}

var _ Transport = &webdavTransport{}

// MakeWebDAVTransport creates a Transport backed by WebDAV.
func MakeWebDAVTransport(cfg *webdav.Config) Transport {
	return &webdavTransport{
		client: webdav.MakeClient(cfg),
	}
}

func (transport *webdavTransport) List(folder string) (files []RemoteFile, err error) {
	resources, err := transport.client.List(folder)
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if resource.IsDir {
			continue
		}

		name := path.Base(resource.Href)
		if !strings.HasSuffix(name, ".org") {
			continue
		}

		files = append(files, RemoteFile{
			Path: resource.Href,
			Name: name,
			ETag: resource.ETag,
			Size: resource.Size,
		})
	}

	return files, nil
}

func (transport *webdavTransport) Read(filePath string) (content []byte, etag string, err error) {
	return transport.client.Get(filePath)
}

func (transport *webdavTransport) Write(filePath string, content []byte, etag string) error {
	return transport.client.Put(filePath, content, etag, "text/plain; charset=utf-8")
}

func (transport *webdavTransport) Delete(filePath string) error {
	return transport.client.Delete(filePath)
}

func (transport *webdavTransport) Close() error {
	return nil
}
