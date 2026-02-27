package remote_http

import (
	"net/http"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

var DefaultRoundTripper http.RoundTripper

func init() {
	DefaultRoundTripper = MakeRoundTripperRetryTimeouts(
		http.DefaultTransport,
		3,
	)
}

func MakeRoundTripperRetry(
	inner http.RoundTripper,
	count int,
	retryFunc func(error) bool,
) RoundTripperRetry {
	return RoundTripperRetry{
		RetryFunc:    retryFunc,
		RetryCount:   count,
		RoundTripper: inner,
	}
}

func MakeRoundTripperRetryTimeouts(
	inner http.RoundTripper,
	count int,
) RoundTripperRetry {
	return RoundTripperRetry{
		RetryFunc:    errors.IsNetTimeout,
		RetryCount:   count,
		RoundTripper: inner,
	}
}

type RoundTripperRetry struct {
	RetryFunc  func(error) bool
	RetryCount int
	http.RoundTripper
}

func (roundTripper RoundTripperRetry) RoundTrip(
	request *http.Request,
) (response *http.Response, err error) {
	count := roundTripper.RetryCount

	for range count {
		if response, err = roundTripper.RoundTripper.RoundTrip(request); err == nil {
			break
		}

		if roundTripper.RetryFunc(err) {
			continue
		}
	}

	return response, err
}
