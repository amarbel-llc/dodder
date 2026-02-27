package sku_json_fmt

import (
	"net/url"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/toml"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
	if err = json.FromTransacted(object, envRepo.GetDefaultBlobStore()); err != nil {
		err = errors.Wrap(err)
		return json, err
	}

	if err = toml.Unmarshal([]byte(json.BlobString), &json.TomlBookmark); err != nil {
		err = errors.Wrapf(err, "%q", json.BlobString)
		return json, err
	}

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
