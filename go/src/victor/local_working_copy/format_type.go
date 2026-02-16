package local_working_copy

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/format"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/sku_lua"
	"code.linenisgreat.com/dodder/go/src/lima/type_blobs"
	"code.linenisgreat.com/dodder/go/src/mike/sku_fmt"
)

type (
	FormatTypeFuncConstructor func(
		*Repo,
		type_blobs.Coder,
		interfaces.WriterAndStringWriter,
	) interfaces.FuncIter[*sku.Transacted]

	FormatTypeFuncConstructorEntry struct {
		Name        string
		description string
		FormatTypeFuncConstructor
	}
)

func makeFormatEntryFromTypeFormat(
	typeEntry FormatTypeFuncConstructorEntry,
) FormatFuncConstructorEntry {
	return FormatFuncConstructorEntry{
		description: typeEntry.description,
		FormatFuncConstructor: func(
			repo *Repo,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			return typeEntry.FormatTypeFuncConstructor(
				repo,
				repo.GetStore().GetTypedBlobStore().Type,
				writer,
			)
		},
	}
}

var typeFormatters = map[string]FormatTypeFuncConstructorEntry{
	"vim-syntax-type": {
		FormatTypeFuncConstructor: func(
			repo *Repo,
			typeBlobStore type_blobs.Coder,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			return func(object *sku.Transacted) (err error) {
				var typeObject *sku.Transacted

				if typeObject, err = repo.GetStore().ReadTypeObject(
					object.GetTypeLock(),
				); err != nil {
					if errors.IsErrNotFound(err) {
						err = nil
					} else {
						err = errors.Wrap(err)
						return err
					}
				}

				if typeObject == nil || typeObject.ObjectId.IsEmpty() ||
					typeObject.GetBlobDigest().IsNull() {
					ty := ""

					switch object.GetGenre() {
					case genres.Type, genres.Tag, genres.Repo, genres.Config:
						ty = "toml"

					default:
						// TODO zettel default typ
					}

					if _, err = fmt.Fprintln(writer, ty); err != nil {
						err = errors.Wrap(err)
						return err
					}

					return err
				}

				var typeBlob type_blobs.Blob
				var repool interfaces.FuncRepool

				if typeBlob, repool, _, err = typeBlobStore.ParseTypedBlob(
					typeObject.GetType(),
					typeObject.GetBlobDigest(),
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				defer repool()

				if _, err = fmt.Fprintln(
					writer,
					typeBlob.GetVimSyntaxType(),
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			}
		},
	},
	"formatters": {
		FormatTypeFuncConstructor: func(
			repo *Repo,
			typeBlobStore type_blobs.Coder,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			return func(object *sku.Transacted) (err error) {
				var typeObject *sku.Transacted

				if typeObject, err = repo.GetStore().ReadTypeObject(
					object.GetTypeLock(),
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				var typeBlob type_blobs.Blob
				var repool interfaces.FuncRepool

				if typeBlob, repool, _, err = typeBlobStore.ParseTypedBlob(
					typeObject.GetType(),
					typeObject.GetBlobDigest(),
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				defer repool()

				lineWriter := format.MakeLineWriter()

				for fn, f := range typeBlob.GetFormatters() {
					fe := f.FileExtension

					if fe == "" {
						fe = fn
					}

					lineWriter.WriteFormat("%s\t%s", fn, fe)
				}

				if _, err = lineWriter.WriteTo(writer); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			}
		},
	},
	"formatter-uti-groups": {
		FormatTypeFuncConstructor: func(
			repo *Repo,
			typeBlobStore type_blobs.Coder,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			format := sku_fmt.MakeFormatterTypeFormatterUTIGroups(
				repo.GetStore().ReadTypeObject,
				typeBlobStore,
			)

			return func(object *sku.Transacted) (err error) {
				if _, err = format.Format(writer, object); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			}
		},
	},
	"hooks.on_pre_commit": {
		FormatTypeFuncConstructor: func(
			repo *Repo,
			typeBlobStore type_blobs.Coder,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			return func(object *sku.Transacted) (err error) {
				var blob type_blobs.Blob
				var repool interfaces.FuncRepool

				if blob, repool, _, err = typeBlobStore.ParseTypedBlob(
					object.GetType(),
					object.GetBlobDigest(),
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				defer repool()

				script := blob.GetStringLuaHooks()

				if script == "" {
					return err
				}

				// TODO switch to typed variant
				var vp sku_lua.LuaVMPoolV1

				if vp, err = repo.GetStore().MakeLuaVMPoolV1(object, script); err != nil {
					err = errors.Wrap(err)
					return err
				}

				vm, vmRepool := vp.GetWithRepool()
				defer vmRepool()

				f := vm.GetField(vm.Top, "on_pre_commit")

				repo.GetUI().Print(f.String())

				return err
			}
		},
	},
}
