//go:build chrest

package store_browser

import (
	"slices"
	"strings"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/echo/descriptions"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type Item struct {
	browser_items.Item
}

func (item *Item) GetExternalObjectId() sku.ExternalObjectId {
	return ids.MakeExternalObjectId(genres.Zettel, item.String())
}

func (item *Item) GetGenre() domain_interfaces.Genre {
	return genres.Zettel
}

func (item *Item) String() string {
	return item.GetKey()
}

func (item *Item) GetKey() string {
	return item.Id.String()
}

func (item *Item) GetType() (t ids.TypeStruct, err error) {
	if err = t.Set("browser-" + item.Id.Type); err != nil {
		err = errors.Wrap(err)
		return t, err
	}

	return t, err
}

// TODO move below to !toml-bookmark type
func (item Item) GetUrlPathTag() (e ids.TagStruct, err error) {
	ur := item.Url.Url()
	els := strings.Split(ur.Hostname(), ".")
	slices.Reverse(els)

	if els[0] == "www" {
		els = els[1:]
	}

	host := strings.Join(els, "-")

	if len(host) == 0 {
		err = errors.ErrorWithStackf("empty host: %q", els)
		return e, err
	}

	if err = e.Set("%zz-site-" + host); err != nil {
		err = errors.Wrap(err)
		return e, err
	}

	return e, err
}

func (item Item) GetTai() (t ids.Tai, err error) {
	if item.Date == "" {
		return t, err
	}

	if err = t.SetFromRFC3339(item.Date); err != nil {
		err = errors.Wrap(err)
		return t, err
	}

	return t, err
}

var errEmptyUrl = newPkgError("empty url")

func (item Item) GetDescription() (b descriptions.Description, err error) {
	if err = b.Set(item.Title); err != nil {
		err = errors.Wrap(err)
		return b, err
	}

	return b, err
}

func (item *Item) WriteToExternal(object *sku.Transacted) (err error) {
	if !item.Id.IsEmpty() {
		if err = object.ExternalObjectId.Set(item.Id.String()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	object.GetMetadataMutable().GetTypeMutable().SetType("!toml-bookmark")

	metadata := object.GetMetadataMutable()

	var tai ids.Tai

	if tai, err = item.GetTai(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	metadata.GetTaiMutable().ResetWith(tai)

	if object.ExternalType, err = item.GetType(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if object.GetMetadata().GetDescription().IsEmpty() {
		if err = object.GetMetadataMutable().GetDescriptionMutable().Set(item.Title); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else if item.Title != "" && object.GetMetadata().GetDescription().String() != item.Title {
		object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
			string_format_writer.Field{
				Key:       "title",
				Value:     item.Title,
				ColorType: string_format_writer.ColorTypeUserData,
			},
		)
	}

	object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
		string_format_writer.Field{
			Key:       "url",
			Value:     item.Url.String(),
			ColorType: string_format_writer.ColorTypeUserData,
		},
	)

	// TODO move to !toml-bookmark type
	var tag ids.TagStruct

	if tag, err = item.GetUrlPathTag(); err == nil {
		if err = metadata.AddTagPtr(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	err = nil

	return err
}

func (item *Item) ReadFromExternal(object *sku.Transacted) (err error) {
	if err = item.Id.Set(
		strings.TrimSuffix(
			object.ExternalObjectId.String(),
			"/",
		),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for field := range object.GetMetadata().GetIndex().GetFields() {
		switch field.Key {
		case "id":
			if field.Value == "" {
				continue
			}

			if err = item.Id.Set(object.ExternalObjectId.String()); err != nil {
				err = errors.Wrap(err)
				return err
			}

		case "", "title":
			item.Title = field.Value

		case "url":
			if err = item.Url.Set(field.Value); err != nil {
				err = errors.Wrap(err)
				return err
			}

		default:
			err = errors.ErrorWithStackf(
				"unsupported field type: %q=%q. Fields: %#v",
				field.Key,
				field.Value,
				object.GetMetadata().GetIndex().GetFields(),
			)

			return err
		}
	}

	// err = todo.Implement()
	return err
}
