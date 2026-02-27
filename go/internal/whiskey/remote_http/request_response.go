package remote_http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/mcp"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"github.com/gorilla/mux"
)

type MethodPath struct {
	Method string
	Path   string
}

type Request struct {
	ctx     errors.Context
	request *http.Request
	MethodPath
	Headers http.Header
	Body    io.ReadCloser
}

func (request Request) Vars() map[string]string {
	return mux.Vars(request.request)
}

type Response struct {
	StatusCode int
	headers    http.Header
	Body       io.ReadCloser
}

func (response *Response) Headers() http.Header {
	if response.headers == nil {
		response.headers = make(http.Header)
	}

	return response.headers
}

func (response *Response) ErrorWithStatus(status int, err error) {
	response.StatusCode = status

	if err != nil {
		var buffer bytes.Buffer

		ui.CLIErrorTreeEncoder.EncodeTo(err, &buffer)
		response.Body = ohio.NopCloser(&buffer)
	}
}

func (response *Response) Error(err error) {
	httpError := errors.Err500InternalServerError

	errors.As(err, &httpError)
	response.ErrorWithStatus(httpError.GetStatusCode(), err)
}

func (response *Response) MCPError(
	status int,
	id any,
	code int,
	message string,
	data any,
) {
	response.StatusCode = status

	mcpResponse := mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	responseBytes, _ := json.Marshal(mcpResponse)
	response.Body = ohio.NopCloser(bytes.NewReader(responseBytes))
}

func ReadErrorFromBody(response *http.Response) (err error) {
	var sb strings.Builder

	if _, err = io.Copy(&sb, response.Body); err != nil {
		err = errors.ErrorWithStackf(
			"failed to read error string from response (%d) body: %q",
			response.StatusCode,
			err,
		)

		return err
	}

	body := sb.String()

	endpointText := fmt.Sprintf(
		"%s %s",
		response.Request.Method,
		response.Request.URL,
	)

	statusText := fmt.Sprintf(
		"%d %s", response.StatusCode,
		http.StatusText(response.StatusCode),
	)

	if body == "" {
		err = errors.BadRequestf(
			"remote responded to request (%q) with error status: %s (error body not provided)",
			endpointText,
			statusText,
		)
	} else {
		err = errors.BadRequestf(
			"remote responded to request (%q) with error (%s):\n\n%s",
			endpointText,
			statusText,
			body,
		)
	}

	return err
}

func ReadErrorFromBodyOnGreaterOrEqual(
	response *http.Response,
	status int,
) (err error) {
	if response.StatusCode < status {
		return err
	}

	err = ReadErrorFromBody(response)

	return err
}

func ReadErrorFromBodyOnNot(
	response *http.Response,
	statuses ...int,
) (err error) {
	if slices.Contains(statuses, response.StatusCode) {
		return err
	}

	err = ReadErrorFromBody(response)

	return err
}
