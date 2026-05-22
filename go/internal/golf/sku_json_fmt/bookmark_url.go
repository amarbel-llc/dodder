package sku_json_fmt

import (
	"io"
	"net/url"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

//go:generate tommy generate
type TomlBookmark struct {
	Url string `toml:"url"`
}

func TomlBookmarkUrl(
	object *sku.Transacted,
	envRepo env_repo.Env,
) (ur *url.URL, err error) {
	var reader mad_domain_interfaces.BlobReader

	if reader, err = envRepo.GetReadBlobStore().MakeBlobReader(object.GetBlobDigest()); err != nil {
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
