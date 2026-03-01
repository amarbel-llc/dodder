package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/lua"
)

type LuaTableV1 struct {
	Transacted   *lua.LTable
	Tags         *lua.LTable
	TagsImplicit *lua.LTable
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
}

func FromLuaTableV1(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV1,
) (err error) {
	transacted := luaTable.Transacted

	genre := genres.MakeOrUnknown(
		luaState.GetField(transacted, "Gattung").String(),
	)

	object.GetObjectIdMutable().SetGenre(genre)
	id := luaState.GetField(transacted, "Kennung").String()

	if id != "" {
		if err = object.GetObjectIdMutable().Set(id); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	tags := luaState.GetField(transacted, "Etiketten")
	tagsTable, ok := tags.(*lua.LTable)

	if !ok {
		err = errors.ErrorWithStackf("expected table but got %T", tags)
		return err
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

	// TODO Bezeichnung
	// TODO Typ
	// TODO Tai
	// TODO Blob
	// TODO Verzeichnisse

	return err
}
