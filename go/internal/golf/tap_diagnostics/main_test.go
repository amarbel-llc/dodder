package tap_diagnostics_test

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/tap_diagnostics"
)

func TestFromErrNotEqual(t *testing.T) {
	var expected, actual markl.Id

	err := markl.ErrNotEqual{
		Expected: &expected,
		Actual:   &actual,
	}

	diag := tap_diagnostics.FromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if _, ok := diag["expected"]; !ok {
		t.Error("expected 'expected' field to be set")
	}
	if _, ok := diag["actual"]; !ok {
		t.Error("expected 'actual' field to be set")
	}
}

func TestFromErrIsNull(t *testing.T) {
	err := markl.ErrIsNull{Purpose: "object-dig"}

	diag := tap_diagnostics.FromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if diag["field"] != "object-dig" {
		t.Errorf("expected field %q, got %q", "object-dig", diag["field"])
	}
}

func TestFromGenericError(t *testing.T) {
	err := fmt.Errorf("something went wrong")

	diag := tap_diagnostics.FromError(err)

	if diag["severity"] != "fail" {
		t.Errorf("expected severity fail, got %q", diag["severity"])
	}
	if diag["message"] != "something went wrong" {
		t.Errorf("expected message, got %q", diag["message"])
	}
}
