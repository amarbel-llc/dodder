package sku_json_fmt

import (
	"bytes"
	"io"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/toml"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

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

	var buffer bytes.Buffer

	if _, err = io.Copy(&buffer, reader); err != nil {
		err = errors.Wrap(err)
		return ur, err
	}

	var tb TomlBookmark

	if err = toml.Unmarshal(buffer.Bytes(), &tb); err != nil {
		err = errors.Wrapf(err, "%q", buffer.String())
		return ur, err
	}

	if ur, err = url.Parse(tb.Url); err != nil {
		err = errors.Wrapf(err, "%q", tb.Url)
		return ur, err
	}

	return ur, err
}
