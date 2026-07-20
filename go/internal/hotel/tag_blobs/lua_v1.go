package tag_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/lib/0/iso_duration"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func MakeLuaSelfApplyV1(
	selfOriginal *sku.Transacted,
) interfaces.FuncIter[*lua.VM] {
	if selfOriginal == nil {
		panic("self was nil")
	}

	self, _ := selfOriginal.CloneTransacted() //repool:owned

	return func(vm *lua.VM) (err error) {
		selfTable, _ := sku_lua.MakeLuaTablePoolV1(vm).GetWithRepool() //repool:owned
		sku_lua.ToLuaTableV1(self, vm.LState, selfTable)
		vm.SetGlobal("Selbst", selfTable.Transacted)
		registerDateHelpers(vm)
		return err
	}
}

// registerDateHelpers exposes the date math the hook VM needs but gopher-lua
// lacks. dodder_advance_date(date, duration) advances a YYYY-MM-DD date by an
// ISO-8601 duration (the PnY nM nW nD subset) and returns the advanced
// YYYY-MM-DD string; on a bad date or duration it raises a lua error. The
// on_commit_fields recurrence hook uses it to roll an actionable object's `due`
// forward when a recurring task is completed.
func registerDateHelpers(vm *lua.VM) {
	vm.SetGlobal("dodder_advance_date", vm.NewFunction(luaAdvanceDate))
}

func luaAdvanceDate(luaState *lua.LState) int {
	date := luaState.ToString(1)
	duration := luaState.ToString(2)

	advanced, err := iso_duration.AdvanceDate(date, duration)
	if err != nil {
		luaState.RaiseError("dodder_advance_date: %s", err)
		return 0
	}

	luaState.Push(lua.LString(advanced))

	return 1
}

type LuaV1 struct {
	sku_lua.LuaVMPoolV1
}

func (a *LuaV1) GetQueryable() sku.Queryable {
	return a
}

func (a *LuaV1) Reset() {
}

func (a *LuaV1) ResetWith(b LuaV1) {
}

func (tb *LuaV1) ContainsSku(tg sku.TransactedGetter) bool {
	// lb := b.luaVMPoolBuilder.Clone().WithApply(MakeSelfApply(sk))
	vm, vmRepool := tb.GetWithRepool()
	defer vmRepool()

	var err error

	var t *lua.LTable

	t, err = vm.VM.GetTopTableOrError()
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	// TODO safer
	f := vm.VM.GetField(t, "contains_sku").(*lua.LFunction)

	tSku, tSkuRepool := vm.TablePool.GetWithRepool()
	defer tSkuRepool()

	vm.VM.Push(f)

	sku_lua.ToLuaTableV1(
		tg,
		vm.VM.LState,
		tSku,
	)

	vm.VM.Push(tSku.Transacted)

	err = vm.VM.PCall(1, 1, nil)
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	retval := vm.LState.Get(1)
	vm.Pop(1)

	if retval.Type() != lua.LTBool {
		ui.Err().Printf("expected bool but got %s", retval.Type())
		return false
	}

	return bool(retval.(lua.LBool))
}
