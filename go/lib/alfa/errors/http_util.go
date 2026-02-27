package errors

import hs "code.linenisgreat.com/dodder/go/lib/_/http_statuses"

func BadRequest(err error) error {
	if Is400BadRequest(err) {
		return err
	}

	return WithoutStack(Err400BadRequest.Wrap(err))
}

func BadRequestf(format string, args ...any) error {
	return WithoutStack(Err400BadRequest.Errorf(format, args...))
}

func BadRequestWrapf(format string, args ...any) error {
	return WithoutStack(Err400BadRequest.Errorf(format, args...))
}

func Is400BadRequest(err error) bool {
	return IsHTTPError(err, hs.Code400BadRequest)
}

func Is499ClientClosedRequest(err error) bool {
	return IsHTTPError(err, hs.Code499ClientClosedRequest)
}

func IsHTTPError(target error, statusCode hs.Code) bool {
	var errHTTP http

	if !As(target, &errHTTP) {
		return false
	}

	return errHTTP.StatusCode == statusCode
}
