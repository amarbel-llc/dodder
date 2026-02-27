package remote_http

import (
	"net/http"
	"net/url"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/values"
)

type UrlData struct {
	Scheme string
	Opaque string        // encoded opaque data
	User   *url.Userinfo // username and password information
	Host   string        // host or host:port (see Hostname and Port methods)
}

func MakeUrlDataFromUri(uri values.Uri) UrlData {
	url := uri.GetUrl()

	return UrlData{
		Scheme: url.Scheme,
		Opaque: url.Opaque,
		User:   url.User,
		Host:   url.Host,
	}
}

func (urlData UrlData) Apply(ur *url.URL) {
	ur.Scheme = urlData.Scheme
	ur.Opaque = urlData.Opaque
	ur.User = urlData.User
	ur.Host = urlData.Host
}

// A round tripper that decorates another round tripper and always populates the
// http requests with given UrlData template.
type RoundTripperHost struct {
	UrlData
	http.RoundTripper
}

func (roundTripper *RoundTripperHost) RoundTrip(
	request *http.Request,
) (response *http.Response, err error) {
	roundTripper.Apply(request.URL)

	if response, err = roundTripper.RoundTripper.RoundTrip(request); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	return response, err
}
