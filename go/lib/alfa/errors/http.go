package errors

import (
	"fmt"

	hs "code.linenisgreat.com/dodder/go/lib/_/http_statuses"
)

func newHttpError(statusCode hs.Code) http {
	return http{StatusCode: statusCode}
}

var (
	Err400BadRequest          = newHttpError(hs.Code400BadRequest)
	Err405MethodNotAllowed    = newHttpError(hs.Code405MethodNotAllowed)
	Err409Conflict            = newHttpError(hs.Code409Conflict)
	Err422UnprocessableEntity = newHttpError(hs.Code409Conflict)
	Err499ClientClosedRequest = newHttpError(hs.Code499ClientClosedRequest)
	Err500InternalServerError = newHttpError(hs.Code500InternalServerError)
	Err501NotImplemented      = newHttpError(hs.Code501NotImplemented)
)

type httpErrDisamb struct{}

type http struct {
	StatusCode hs.Code
	exposeHTTP bool
	underlying error
}

func (err http) GetStatusCode() int {
	return int(err.StatusCode)
}

func (err http) Error() string {
	return fmt.Sprintf("errors.HTTP: %s", err.StatusCode.String())
}

func (err http) Is(target error) bool {
	_, ok := target.(http)
	return ok
}

func (err http) Unwrap() error {
	return err.underlying
}

func (err http) WithStack() error {
	return WrapSkip(1, err)
}

func (err http) WrapIncludingHTTP(underlying error) http {
	return http{
		StatusCode: err.StatusCode,
		underlying: underlying,
		exposeHTTP: true,
	}
}

func (err http) Wrap(underlying error) http {
	return http{
		StatusCode: err.StatusCode,
		underlying: underlying,
	}
}

// Creates a new error from `format` and `args` a returns a new HTTP error that
// wraps it, but does not expose the HTTP error's message.
//
// The returned error will satisfy the appropriate `IsHTTPError(err, status)`
// call, but when using `error_coders` to print it, but it won't show the HTTP
// error
func (err http) Errorf(format string, args ...any) http {
	err = err.Wrap(fmt.Errorf(format, args...))
	return err
}

func (err http) ShouldHideUnwrap() bool {
	return !err.exposeHTTP
}

func (err http) GetErrorType() httpErrDisamb {
	return httpErrDisamb{}
}
