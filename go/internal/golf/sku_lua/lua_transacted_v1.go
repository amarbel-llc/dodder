package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type LuaTableV1 struct {
	Transacted   *lua.LTable
	Tags         *lua.LTable
	TagsImplicit *lua.LTable
	Fields       *lua.LTable
}

func ToLuaTableV1(
	tg sku.TransactedGetter,
	luaState *lua.LState,
	luaTable *LuaTableV1,
) {
	object := tg.GetSku()

	luaState.SetField(
		luaTable.Transacted,
		"Gattung",
		lua.LString(object.GetGenre().String()),
	)
	luaState.SetField(
		luaTable.Transacted,
		"Kennung",
		lua.LString(object.GetObjectId().String()),
	)
	luaState.SetField(
		luaTable.Transacted,
		"Typ",
		lua.LString(object.GetType().String()),
	)

	tags := luaTable.Tags

	for tag := range object.GetMetadata().AllTags() {
		luaState.SetField(tags, tag.String(), lua.LBool(true))
	}

	tags = luaTable.TagsImplicit

	for tag := range object.GetMetadata().GetIndex().GetImplicitTags().All() {
		luaState.SetField(tags, tag.String(), lua.LBool(true))
	}

	// Project the metadata index fields (name -> string value) so hooks can
	// read them as kinder.Fields.<name> and mutate them in place; FromLuaTableV1
	// reads the mutated values back (RFC 0006 Phase 1 field write-back).
	fieldsTable := luaTable.Fields

	for field := range object.GetMetadata().GetIndex().GetFields() {
		luaState.SetField(fieldsTable, field.Key, lua.LString(field.Value))
	}
}

// FromLuaTableV1 writes the hook's mutations back onto object. It returns
// fieldsChanged when the hook altered any projected field value (RFC 0006
// Phase 1 field write-back), so the commit pipeline can run its single bounded,
// hook-free write-back pass. Tag and id write-back are unconditional and do not
// affect fieldsChanged.
func FromLuaTableV1(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV1,
) (fieldsChanged bool, err error) {
	transacted := luaTable.Transacted

	genre := genres.MakeOrUnknown(
		luaState.GetField(transacted, "Gattung").String(),
	)

	object.GetObjectIdMutable().SetGenre(genre)
	id := luaState.GetField(transacted, "Kennung").String()

	if id != "" {
		if err = object.GetObjectIdMutable().Set(id); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}
	}

	tags := luaState.GetField(transacted, "Etiketten")
	tagsTable, ok := tags.(*lua.LTable)

	if !ok {
		err = errors.ErrorWithStackf("expected table but got %T", tags)
		return fieldsChanged, err
	}

	object.GetMetadataMutable().ResetTags()

	tagsTable.ForEach(
		func(key, value lua.LValue) {
			var tag ids.TagStruct

			if err = tag.Set(key.String()); err != nil {
				err = errors.Wrap(err)
				panic(err)
			}

			errors.PanicIfError(object.GetMetadataMutable().AddTagPtr(tag))
		},
	)

	fieldsChanged = writeFieldsBack(object, luaTable.Fields)

	// TODO Bezeichnung
	// TODO Typ
	// TODO Tai
	// TODO Blob
	// TODO Verzeichnisse

	return fieldsChanged, err
}

// writeFieldsBack applies any values the hook set on kinder.Fields back onto
// the object's projected index fields. Only fields that already exist in the
// index are updated (Key + TypeBlobDigest preserved); a brand-new key is
// ignored because a field with no TypeBlobDigest cannot be persisted by the
// fields-writer. A nil or empty Fields table is a graceful no-op — this matters
// because FromLuaTableV1 also runs for on_pre_commit, before fields are
// projected. Returns whether any field value actually changed.
func writeFieldsBack(
	object *sku.Transacted,
	fieldsTable *lua.LTable,
) (fieldsChanged bool) {
	if fieldsTable == nil {
		return fieldsChanged
	}

	indexFields := object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable()

	if indexFields.Len() == 0 {
		return fieldsChanged
	}

	fieldsTable.ForEach(
		func(key, value lua.LValue) {
			keyString := key.String()
			valueString := value.String()

			for i := 0; i < indexFields.Len(); i++ {
				field := indexFields.At(i)

				if field.Key != keyString {
					continue
				}

				if field.Value == valueString {
					break
				}

				field.Value = valueString
				(*indexFields)[i] = field
				fieldsChanged = true

				break
			}
		},
	)

	return fieldsChanged
}
