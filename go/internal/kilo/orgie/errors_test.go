package orgie

import (
	"errors"
	"strings"
	"testing"
)

// TestErrOrganizeBaseMissingMessageMentionsRegenerate pins the
// dodder#374(b) plan §8 cold-cutover error's guidance text: a document
// with no `_base` field must tell the user how to recover (regenerate),
// not just that it's invalid.
func TestErrOrganizeBaseMissingMessageMentionsRegenerate(t1 *testing.T) {
	err := ErrOrganizeBaseMissing{}

	if !strings.Contains(err.Error(), "_base") {
		t1.Errorf("expected error message to mention `_base`, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "dodder organize") {
		t1.Errorf("expected error message to include the regenerate command, got: %s", err.Error())
	}
}

// TestErrOrganizeBaseMissingIsMatchesOwnTypeOnly pins the typed-sentinel
// convention this package uses (Is checks type identity, not value
// equality) -- errors.Is must match another ErrOrganizeBaseMissing but
// not an unrelated error.
func TestErrOrganizeBaseMissingIsMatchesOwnTypeOnly(t1 *testing.T) {
	err := ErrOrganizeBaseMissing{}

	if !errors.Is(err, ErrOrganizeBaseMissing{}) {
		t1.Errorf("expected errors.Is to match another ErrOrganizeBaseMissing")
	}

	if errors.Is(err, ErrBaseUndereferenceable{}) {
		t1.Errorf("expected errors.Is to NOT match an unrelated error type")
	}
}

// TestErrBaseUndereferenceableMessageIncludesDigestAndCause pins the
// dodder#374(b) plan §3 error's content: the digest that couldn't be
// dereferenced and the underlying cause must both be visible, so a user
// (or bats assertion) can tell this apart from ErrOrganizeBaseMissing
// ("no _base" vs "_base present but broken") and diagnose the cause.
func TestErrBaseUndereferenceableMessageIncludesDigestAndCause(t1 *testing.T) {
	cause := errors.New("blob not found")

	err := ErrBaseUndereferenceable{
		Digest: "blake2b256-deadbeef",
		Cause:  cause,
	}

	message := err.Error()

	if !strings.Contains(message, "blake2b256-deadbeef") {
		t1.Errorf("expected error message to include the digest, got: %s", message)
	}

	if !strings.Contains(message, "blob not found") {
		t1.Errorf("expected error message to include the cause, got: %s", message)
	}
}

// TestErrBaseUndereferenceableIsMatchesOwnTypeOnlyIgnoringFields pins that
// Is compares TYPE identity, not field values (Digest/Cause vary per
// occurrence but errors.Is must still recognize any instance as "the same
// kind of error").
func TestErrBaseUndereferenceableIsMatchesOwnTypeOnlyIgnoringFields(t1 *testing.T) {
	err := ErrBaseUndereferenceable{Digest: "a", Cause: errors.New("x")}

	if !errors.Is(err, ErrBaseUndereferenceable{Digest: "different", Cause: errors.New("y")}) {
		t1.Errorf("expected errors.Is to match another ErrBaseUndereferenceable regardless of field values")
	}

	if errors.Is(err, ErrOrganizeBaseMissing{}) {
		t1.Errorf("expected errors.Is to NOT match an unrelated error type")
	}
}
