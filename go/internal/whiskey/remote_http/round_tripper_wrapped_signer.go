package remote_http

import (
	"bytes"
	"net/http"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

const (
	headerChallengeNonce    = "X-Dodder-Challenge-Nonce"
	headerChallengeResponse = "X-Dodder-Challenge-Response"
	headerRepoPublicKey     = "X-Dodder-Repo-Public_Key"
	headerRepoSig           = "X-Dodder-Repo-Sig"
)

type RoundTripperBufioWrappedSigner struct {
	PublicKey domain_interfaces.MarklId
	roundTripperBufio
}

// TODO extract signing into an agnostic middleware
func (roundTripper *RoundTripperBufioWrappedSigner) RoundTrip(
	request *http.Request,
) (response *http.Response, err error) {
	var nonce markl.Id

	if err = nonce.GeneratePrivateKey(
		nil,
		markl.FormatIdNonceSec,
		markl.PurposeRequestAuthChallengeV1,
	); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	request.Header.Add(headerChallengeNonce, nonce.String())

	if response, err = roundTripper.roundTripperBufio.RoundTrip(
		request,
	); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	sigString := response.Header.Get(headerChallengeResponse)

	if sigString == "" {
		err = errors.Errorf("signature empty or not provided")
		return response, err
	}

	var sig markl.Id

	if err = sig.Set(sigString); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	pubkeyString := response.Header.Get(headerRepoPublicKey)

	var pubkey markl.Id

	if err = pubkey.Set(
		pubkeyString,
	); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	if roundTripper.PublicKey.IsNull() {
		// TODO present prompt to user for TOFU
	} else {
		if !bytes.Equal(roundTripper.PublicKey.GetBytes(), pubkey.GetBytes()) {
			err = errors.Errorf(
				"expected pubkey %q but got %q",
				roundTripper.PublicKey.GetBytes(),
				pubkey.GetBytes(),
			)

			return response, err
		}
	}

	if err = pubkey.Verify(
		nonce,
		sig,
	); err != nil {
		err = errors.Wrap(err)
		return response, err
	}

	return response, err
}
