//go:build test

package sku

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// CloseEmpty on a genuinely empty list is a plain close: the discard
// sites (store.Initialize's re-init guard, inventory_list_store.Create's
// empty-list path) must keep working for the case they exist for.
func TestWorkingListCloseEmptyOnEmptyList(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeWorkingList(nil, nil, nil)

	if err := sut.CloseEmpty(); err != nil {
		t.Errorf("CloseEmpty on empty list: %s", err)
	}
}

// CloseEmpty with pending records must refuse loudly: a discard here
// would silently drop the records from inventory-list history
// (dodder#369). count is set directly rather than via Add so the test
// needs no coder or blob writer — the guard reads only Len().
func TestWorkingListCloseEmptyRefusesPendingRecords(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeWorkingList(nil, nil, nil)
	sut.count = 3

	err := sut.CloseEmpty()
	if err == nil {
		t.Errorf("CloseEmpty with pending records: expected an error, got nil")
	}
}
