package sku_json_fmt

import (
	"io"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

//go:generate tommy generate
type TomlBookmark struct {
	Url string `toml:"url"`
}

func TomlBookmarkUrl(
	object *sku.Transacted,
	envRepo env_repo.Env,
) (ur *url.URL, err error) {
	var reader domain_interfaces.BlobReader

	if reader, err = envRepo.GetDefaultBlobStore().MakeBlobReader(object.GetBlobDigest()); err != nil {
		err = errors.Wrap(err)
		return ur, err
	}

	defer errors.DeferredCloser(&err, reader)

	var b []byte

	if b, err = io.ReadAll(reader); err != nil {
		err = errors.Wrap(err)
		return ur, err
	}

	doc, decErr := DecodeTomlBookmark(b)
	if decErr != nil {
		err = errors.Wrapf(decErr, "%q", string(b))
		return ur, err
	}

	tb := doc.Data()

	if ur, err = url.Parse(tb.Url); err != nil {
		err = errors.Wrapf(err, "%q", tb.Url)
		return ur, err
	}

	return ur, err
}
