package orgie

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrorRead struct {
	error

	line, column int
}

func (err ErrorRead) Is(target error) bool {
	_, ok := target.(ErrorRead)
	return ok
}

func (err ErrorRead) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

// ErrOrganizeBaseMissing is dodder#374(b) plan §8's cold-cutover error:
// a document with no `_base` field is invalid, full stop -- organize
// documents are ephemeral action, not durable artifacts, so there is
// no legacy fallback to fall back to.
type ErrOrganizeBaseMissing struct{}

func (err ErrOrganizeBaseMissing) Error() string {
	return "this organize document has no `_base` field.\n\n" +
		"organize documents are ephemeral action, not durable artifacts -- " +
		"edits can only be applied against the exact document `organize` " +
		"generated. Regenerate with:\n\n" +
		"    dodder organize <your original query>\n\n" +
		"then make your edits in the freshly generated document."
}

func (err ErrOrganizeBaseMissing) Is(target error) bool {
	_, ok := target.(ErrOrganizeBaseMissing)
	return ok
}

func (err ErrOrganizeBaseMissing) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

// ErrBaseUndereferenceable is dodder#374(b) plan §3: `_base` is present
// and well-formed but the digest it names can't be read back (a
// remote peer's copy of the repo, a blob store that was never synced,
// or manual blob-store surgery -- not a GC concern, since dodder has
// no blob-GC mechanism today). Distinct from ErrOrganizeBaseMissing:
// "you have a `_base`, but it's stale/wrong" vs "you have no `_base`
// at all" -- different failure points, same regenerate guidance.
type ErrBaseUndereferenceable struct {
	Digest string
	Cause  error
}

func (err ErrBaseUndereferenceable) Error() string {
	return fmt.Sprintf(
		"this organize document's `_base=@%s` field could not be "+
			"dereferenced (%s).\n\n"+
			"organize documents are ephemeral action, not durable artifacts -- "+
			"edits can only be applied against the exact document `organize` "+
			"generated. Regenerate with:\n\n"+
			"    dodder organize <your original query>\n\n"+
			"then make your edits in the freshly generated document.",
		err.Digest,
		err.Cause,
	)
}

func (err ErrBaseUndereferenceable) Is(target error) bool {
	_, ok := target.(ErrBaseUndereferenceable)
	return ok
}

func (err ErrBaseUndereferenceable) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

// ErrConflicts is dodder#374(b) plan §4: one or more objects changed both
// in the patch and independently in the live store since `_base` was
// generated, on tags ComputeThreeWay's conservative object-level pass
// can't prove are the same edit -- "v1 rejects loudly (mergetool
// deferred)" rather than silently picking a side. ObjectIds is the
// conflicted set, for a caller to report or (eventually) hand to an
// interactive resolver.
type ErrConflicts struct {
	ObjectIds []string
}

func (err ErrConflicts) Error() string {
	return fmt.Sprintf(
		"%d object(s) changed both in your edit and in the store since "+
			"this document was generated, and can't be merged "+
			"automatically:\n  %s\n\n"+
			"Regenerate with `dodder organize <your original query>` and "+
			"re-apply your edits.",
		len(err.ObjectIds),
		strings.Join(err.ObjectIds, "\n  "),
	)
}

func (err ErrConflicts) Is(target error) bool {
	_, ok := target.(ErrConflicts)
	return ok
}

func (err ErrConflicts) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

var (
	_ error = ErrOrganizeBaseMissing{}
	_ error = ErrBaseUndereferenceable{}
	_ error = ErrConflicts{}
)
