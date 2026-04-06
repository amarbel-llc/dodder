package webdav

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

const requestTimeout = 30 * time.Second

// Config holds WebDAV connection parameters.
type Config struct {
	URL      string
	Username string
	Password string
}

// Resource represents a file or collection returned by PROPFIND.
type Resource struct {
	Href        string
	DisplayName string
	ContentType string
	ETag        string
	Size        int64
	IsDir       bool
}

// Client is a WebDAV HTTP client for file-level operations.
type Client struct {
	cfg  *Config
	http *http.Client
}

// MakeClient creates a WebDAV client from the given configuration.
func MakeClient(cfg *Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (client *Client) do(
	method string,
	url string,
	body io.Reader,
	headers map[string]string,
) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("webdav request: %w", err)
	}

	if client.cfg.Username != "" {
		req.SetBasicAuth(client.cfg.Username, client.cfg.Password)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return client.http.Do(req)
}

// List performs a PROPFIND on the given collection URL and returns the
// contained resources.
func (client *Client) List(collectionURL string) (resources []Resource, err error) {
	propfindBody := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:displayname/>
    <d:getcontenttype/>
    <d:getetag/>
    <d:getcontentlength/>
    <d:resourcetype/>
  </d:prop>
</d:propfind>`

	resp, err := client.do("PROPFIND", collectionURL, strings.NewReader(propfindBody), map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
		"Depth":        "1",
	})
	if err != nil {
		return nil, fmt.Errorf("PROPFIND %s: %w", collectionURL, err)
	}
	defer errors.DeferredCloser(&err, resp.Body)

	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("PROPFIND %s: status %d", collectionURL, resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND read body: %w", err)
	}

	return parsePropfindResponse(responseBody, collectionURL)
}

// Get retrieves the content of a file at the given URL.
func (client *Client) Get(fileURL string) (content []byte, etag string, err error) {
	resp, err := client.do("GET", fileURL, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("GET %s: %w", fileURL, err)
	}
	defer errors.DeferredCloser(&err, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GET %s: status %d", fileURL, resp.StatusCode)
	}

	content, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("GET read body: %w", err)
	}

	etag = resp.Header.Get("ETag")
	return content, etag, nil
}

// Put creates or updates a file at the given URL. If etag is non-empty, the
// request uses If-Match for conditional update. The contentType parameter
// sets the Content-Type header; if empty, defaults to
// "application/octet-stream".
func (client *Client) Put(fileURL string, content []byte, etag string, contentType string) (err error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	headers := map[string]string{
		"Content-Type": contentType,
	}

	if etag != "" {
		headers["If-Match"] = etag
	}

	resp, err := client.do("PUT", fileURL, bytes.NewReader(content), headers)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", fileURL, err)
	}
	defer errors.DeferredCloser(&err, resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PUT %s: status %d", fileURL, resp.StatusCode)
	}

	return nil
}

// Delete removes the resource at the given URL.
func (client *Client) Delete(fileURL string) (err error) {
	resp, err := client.do("DELETE", fileURL, nil, nil)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", fileURL, err)
	}
	defer errors.DeferredCloser(&err, resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE %s: status %d", fileURL, resp.StatusCode)
	}

	return nil
}

// Mkcol creates a new collection (directory) at the given URL.
func (client *Client) Mkcol(collectionURL string) (err error) {
	resp, err := client.do("MKCOL", collectionURL, nil, nil)
	if err != nil {
		return fmt.Errorf("MKCOL %s: %w", collectionURL, err)
	}
	defer errors.DeferredCloser(&err, resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MKCOL %s: status %d", collectionURL, resp.StatusCode)
	}

	return nil
}

// --- XML types for WebDAV PROPFIND responses ---

type multistatusResponse struct {
	XMLName   xml.Name      `xml:"DAV: multistatus"`
	Responses []davResponse `xml:"DAV: response"`
}

type davResponse struct {
	Href     string     `xml:"DAV: href"`
	Propstat []propstat `xml:"DAV: propstat"`
}

type propstat struct {
	Prop   prop   `xml:"DAV: prop"`
	Status string `xml:"DAV: status"`
}

type prop struct {
	DisplayName   string       `xml:"DAV: displayname"`
	ContentType   string       `xml:"DAV: getcontenttype"`
	ETag          string       `xml:"DAV: getetag"`
	ContentLength int64        `xml:"DAV: getcontentlength"`
	ResourceType  resourceType `xml:"DAV: resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"DAV: collection"`
}

func parsePropfindResponse(body []byte, collectionURL string) (resources []Resource, err error) {
	var ms multistatusResponse
	if err = xml.Unmarshal(body, &ms); err != nil {
		return nil, fmt.Errorf("parse PROPFIND response: %w", err)
	}

	// Normalize the collection URL for comparison.
	collectionPath := normalizeHref(collectionURL)

	for _, resp := range ms.Responses {
		href := normalizeHref(resp.Href)

		// Skip the collection itself.
		if href == collectionPath {
			continue
		}

		resource := Resource{Href: resp.Href}

		for _, ps := range resp.Propstat {
			if !strings.Contains(ps.Status, "200") {
				continue
			}

			resource.DisplayName = ps.Prop.DisplayName
			resource.ContentType = ps.Prop.ContentType
			resource.ETag = ps.Prop.ETag
			resource.Size = ps.Prop.ContentLength
			resource.IsDir = ps.Prop.ResourceType.Collection != nil
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// normalizeHref strips scheme+host and trailing slashes for comparison.
func normalizeHref(href string) string {
	// Strip scheme://host if present.
	if idx := strings.Index(href, "://"); idx >= 0 {
		rest := href[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
			href = rest[slashIdx:]
		}
	}

	return strings.TrimRight(href, "/")
}
