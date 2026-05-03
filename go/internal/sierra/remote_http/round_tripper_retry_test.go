package remote_http

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

type fakeRoundTripper struct {
	calls    int
	scripted []roundTripResult
}

type roundTripResult struct {
	response *http.Response
	err      error
}

func (rt *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := rt.calls
	rt.calls++

	if idx >= len(rt.scripted) {
		return nil, errors.New("fakeRoundTripper: ran out of scripted results")
	}

	r := rt.scripted[idx]
	return r.response, r.err
}

func makeOKResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
	}
}

type fakeNetTimeoutError struct{}

func (fakeNetTimeoutError) Error() string   { return "fake net timeout" }
func (fakeNetTimeoutError) Timeout() bool   { return true }
func (fakeNetTimeoutError) Temporary() bool { return true }

var _ net.Error = fakeNetTimeoutError{}

type fakeNonTimeoutNetError struct{}

func (fakeNonTimeoutNetError) Error() string   { return "fake non-timeout net error" }
func (fakeNonTimeoutNetError) Timeout() bool   { return false }
func (fakeNonTimeoutNetError) Temporary() bool { return false }

var _ net.Error = fakeNonTimeoutNetError{}

func TestRoundTripperRetrySuccessFirstTry(t1 *testing.T) {
	t := ui.T{T: t1}

	okResp := makeOKResponse()
	inner := &fakeRoundTripper{
		scripted: []roundTripResult{
			{response: okResp, err: nil},
		},
	}

	rt := MakeRoundTripperRetry(inner, 3, func(error) bool { return true })

	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if err != nil {
		t.Fatalf("unexpected error building request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if resp != okResp {
		t.Fatalf("expected scripted ok response, got: %v", resp)
	}
	if inner.calls != 1 {
		t.Fatalf("expected exactly 1 call (success short-circuits), got: %d", inner.calls)
	}
}

func TestRoundTripperRetryRetriableErrorThenSuccess(t1 *testing.T) {
	t := ui.T{T: t1}

	okResp := makeOKResponse()
	inner := &fakeRoundTripper{
		scripted: []roundTripResult{
			{response: nil, err: fakeNetTimeoutError{}},
			{response: nil, err: fakeNetTimeoutError{}},
			{response: okResp, err: nil},
		},
	}

	predicate := func(err error) bool {
		var netErr net.Error
		return errors.As(err, &netErr) && netErr.Timeout()
	}

	rt := MakeRoundTripperRetry(inner, 5, predicate)

	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if err != nil {
		t.Fatalf("unexpected error building request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected nil error after retries, got: %v", err)
	}
	if resp != okResp {
		t.Fatalf("expected scripted ok response, got: %v", resp)
	}
	if inner.calls != 3 {
		t.Fatalf("expected exactly 3 calls (2 retries + success), got: %d", inner.calls)
	}
}

func TestRoundTripperRetryNonRetriableErrorBreaksEarly(t1 *testing.T) {
	t := ui.T{T: t1}

	nonRetriable := fakeNonTimeoutNetError{}
	inner := &fakeRoundTripper{
		scripted: []roundTripResult{
			{response: nil, err: nonRetriable},
			{response: nil, err: errors.New("should never be called")},
			{response: nil, err: errors.New("should never be called")},
		},
	}

	predicate := func(err error) bool {
		var netErr net.Error
		return errors.As(err, &netErr) && netErr.Timeout()
	}

	rt := MakeRoundTripperRetry(inner, 5, predicate)

	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if err != nil {
		t.Fatalf("unexpected error building request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error to propagate, got nil")
	}
	if !errors.Is(err, nonRetriable) && err.Error() != nonRetriable.Error() {
		t.Fatalf("expected non-retriable error to propagate verbatim, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got: %v", resp)
	}
	if inner.calls != 1 {
		t.Fatalf("expected exactly 1 call (non-retriable breaks early), got: %d", inner.calls)
	}
}

func TestRoundTripperRetryExhaustionReturnsLastError(t1 *testing.T) {
	t := ui.T{T: t1}

	inner := &fakeRoundTripper{
		scripted: []roundTripResult{
			{response: nil, err: fakeNetTimeoutError{}},
			{response: nil, err: fakeNetTimeoutError{}},
			{response: nil, err: fakeNetTimeoutError{}},
		},
	}

	predicate := func(err error) bool {
		var netErr net.Error
		return errors.As(err, &netErr) && netErr.Timeout()
	}

	rt := MakeRoundTripperRetry(inner, 3, predicate)

	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if err != nil {
		t.Fatalf("unexpected error building request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error after exhausting retries, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got: %v", resp)
	}
	if inner.calls != 3 {
		t.Fatalf("expected exactly RetryCount=3 calls, got: %d", inner.calls)
	}
}

func TestMakeRoundTripperRetryTimeoutsUsesIsNetTimeout(t1 *testing.T) {
	t := ui.T{T: t1}

	inner := &fakeRoundTripper{
		scripted: []roundTripResult{
			{response: nil, err: fakeNonTimeoutNetError{}},
			{response: nil, err: errors.New("should never be called")},
		},
	}

	rt := MakeRoundTripperRetryTimeouts(inner, 5)

	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if err != nil {
		t.Fatalf("unexpected error building request: %v", err)
	}

	_, err = rt.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected non-timeout error to propagate, got nil")
	}
	if inner.calls != 1 {
		t.Fatalf("expected exactly 1 call (non-timeout error not retriable), got: %d", inner.calls)
	}
}
