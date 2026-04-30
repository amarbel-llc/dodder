package remote_http

import (
	"bytes"
	"io"
	"net/http"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

const (
	headerChallengeNonce    = "X-Dodder-Challenge-Nonce"
	headerChallengeResponse = "X-Dodder-Challenge-Response"
	headerRepoPublicKey     = "X-Dodder-Repo-Public_Key"
	headerRepoSig           = "X-Dodder-Repo-Sig"
)

type RoundTripperBufioWrappedSigner struct {
	PublicKey  domain_interfaces.MarklId
	HashFormat domain_interfaces.FormatHash
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

	if roundTripper.HashFormat != nil && response.Body != nil {
		hash, repoolHash := roundTripper.HashFormat.GetHash()

		response.Body = &verifyingBodyReader{
			body:       response.Body,
			response:   response,
			hash:       hash,
			repoolHash: repoolHash,
			pubkey:     pubkey,
		}
	}

	return response, err
}

type verifyingBodyReader struct {
	body       io.ReadCloser
	response   *http.Response
	hash       domain_interfaces.Hash
	repoolHash interfaces.FuncRepool
	pubkey     markl.Id
	verified   bool
}

func (r *verifyingBodyReader) Read(p []byte) (n int, err error) {
	n, err = r.body.Read(p)

	if n > 0 {
		r.hash.Write(p[:n])
	}

	if errors.IsEOF(err) && !r.verified {
		r.verified = true

		if verifyErr := r.verifyTrailer(); verifyErr != nil {
			return n, verifyErr
		}
	}

	return n, err
}

func (r *verifyingBodyReader) Close() error {
	if r.repoolHash != nil {
		defer r.repoolHash()
	}

	return r.body.Close()
}

func (r *verifyingBodyReader) verifyTrailer() error {
	sigString := r.response.Trailer.Get(headerRepoSig)

	if sigString == "" {
		return errors.Errorf("response body signature trailer missing")
	}

	var sig markl.Id

	if err := sig.Set(sigString); err != nil {
		return errors.Wrap(err)
	}

	bodyDigest, repoolDigest := r.hash.GetMarklId()
	defer repoolDigest()

	if err := r.pubkey.Verify(bodyDigest, sig); err != nil {
		return errors.Wrapf(err, "response body signature verification failed")
	}

	return nil
}
