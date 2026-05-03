package remote_http

import (
	"io"
	"net/http"
)

func (client *client) newRequest(
	method, url string,
	body io.Reader,
) (*http.Request, error) {
	return http.NewRequestWithContext(client.GetEnv(), method, url, body)
}
