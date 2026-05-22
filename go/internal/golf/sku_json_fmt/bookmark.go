package sku_json_fmt

import (
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type JsonWithUrl struct {
	Transacted
	TomlBookmark
}

func MakeJsonTomlBookmark(
	object *sku.Transacted,
	envRepo env_repo.Env,
	tabs []any,
) (json JsonWithUrl, err error) {
	if err = json.FromTransacted(object, envRepo.GetReadBlobStore()); err != nil {
		err = errors.Wrap(err)
		return json, err
	}

	doc, decErr := DecodeTomlBookmark([]byte(json.BlobString))
	if decErr != nil {
		err = errors.Wrapf(decErr, "%q", json.BlobString)
		return json, err
	}

	json.TomlBookmark = *doc.Data()

	if _, err = url.Parse(json.Url); err != nil {
		err = errors.Wrap(err)
		return json, err
	}

	for _, tabRaw := range tabs {
		tab := tabRaw.(map[string]any)

		if _, err = url.Parse(tab["url"].(string)); err != nil {
			err = errors.Wrap(err)
			return json, err
		}
	}

	return json, err
}
