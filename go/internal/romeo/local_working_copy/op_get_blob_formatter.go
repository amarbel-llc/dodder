package local_working_copy

import (
	"maps"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// TODO add support for checked out types
func (local *Repo) GetBlobFormatter(
	typeObject *sku.Transacted,
	formatId string,
	utiGroup string,
) (blobFormatter script_config.RemoteScript, err error) {
	if typeObject.GetType().IsEmpty() {
		err = errors.ErrorWithStackf("empty type")
		return blobFormatter, err
	}

	var typeBlob type_blobs.Blob
	var repool interfaces.FuncRepool

	if typeBlob, repool, _, err = local.GetStore().GetTypedBlobStore().Type.ParseTypedBlob(
		typeObject.GetType(),
		typeObject.GetBlobDigest(),
	); err != nil {
		if repool != nil {
			repool()
		}

		err = errors.Wrap(err)
		return blobFormatter, err
	}

	defer repool()

	ok := false

	if utiGroup == "" {
		getBlobFormatter := func(formatId string) script_config.RemoteScript {
			var formatIds []string

			if formatId == "" {
				formatIds = []string{"text-edit", "text"}
			} else {
				formatIds = []string{formatId}
			}

			for _, formatId := range formatIds {
				blobFormatter, ok = typeBlob.GetFormatters()[formatId]

				if ok {
					return blobFormatter
				}
			}

			return nil
		}

		blobFormatter = getBlobFormatter(formatId)

		return blobFormatter, err
	}

	var g type_blobs.UTIGroup
	g, ok = typeBlob.GetFormatterUTIGroups()[utiGroup]

	if !ok {
		err = errors.BadRequestf(
			"no uti group: %q. Available groups: %s",
			utiGroup,
			slices.Collect(maps.Keys(typeBlob.GetFormatterUTIGroups())),
		)
		return blobFormatter, err
	}

	ft, ok := g.Map()[formatId]

	if !ok {
		err = errors.ErrorWithStackf(
			"no format id %q for uti group %q. Available groups: %s",
			formatId,
			utiGroup,
			slices.Collect(maps.Keys(g.Map())),
		)

		return blobFormatter, err
	}

	formatId = ft

	blobFormatter, ok = typeBlob.GetFormatters()[formatId]

	if !ok {
		ui.Err().Print("no matching format id")
		blobFormatter = nil
		// TODO-P2 allow option to error on missing format
		// err = errors.Normalf("no format id %q", actualFormatId)
		// return

		return blobFormatter, err
	}

	return blobFormatter, err
}
