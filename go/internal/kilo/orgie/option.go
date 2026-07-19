package orgie

import (
	"fmt"
	"strings"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"

	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/values"
)

type OptionComment interface {
	CloneOptionComment() OptionComment
	interfaces.StringerSetter
}

type OptionCommentWithApply interface {
	OptionComment
	ApplyToText(Options, *Assignment) error
	ApplyToReader(Options, *reader) error
	ApplyToWriter(Options, *writer) error
}

// OptionCommentSettingsField marks an OptionComment as a `_`-reserved
// document settings field (dodder#374, cutting-garden RFC 0015) rather than
// a `%` comment. Implementing this is the only step needed to migrate an
// OptionComment's write-side spelling -- Metadata.WriteTo checks it
// generically, matching ReadFrom's already-generic prototype-keyed parsing,
// so no new type-switch branch is needed per migrated setting.
type OptionCommentSettingsField interface {
	OptionComment
	IsSettingsField() bool
}

// isSettingsField reports whether a single-wrapped OptionCommentWithKey
// (from OptionCommentSet's prototype registry or OptionComments slice)
// should use the `- _key=value` settings-field spelling rather than
// `% key:value`. Shared by Metadata.ReadFrom (deciding whether a `-
// _key=value` line is a recognized settings field or falls through to
// ordinary tag parsing) and Metadata.WriteTo (deciding how to emit an
// already-registered OptionComment), so the two directions can't drift.
func isSettingsField(o OptionComment) bool {
	ocwk, ok := o.(OptionCommentWithKey)
	if !ok {
		return false
	}

	sf, ok := ocwk.OptionComment.(OptionCommentSettingsField)

	return ok && sf.IsSettingsField()
}

// TODO add config to automatically add dry run if necessary
func MakeOptionCommentSet(
	elements map[string]OptionComment,
	options ...OptionComment,
) OptionCommentSet {
	ocs := OptionCommentSet{
		prototype:      make(PrototypeOptionComments),
		OptionComments: options,
	}

	if elements != nil {
		for k, el := range elements {
			ocs.AddPrototype(k, el)
		}
	}

	ocs.AddPrototype("hide", optionCommentHide(""))
	ocs.AddPrototype("", optionCommentHide(""))

	// `_base`/`_allow-deletion` (dodder#374(b), cutting-garden RFC 0015):
	// registered unconditionally, unlike `_dry-run` (which only becomes a
	// registered prototype when the CLI's `-dry-run` flag is active,
	// ApplyToOrganizeOptions). `_base` is required on every organize
	// document and `_allow-deletion` must always be settable by hand --
	// neither mirrors ambient CLI/config state the way dry-run does.
	ocs.AddPrototype("base", &OptionCommentBaseDigest{})
	ocs.AddPrototype("allow-deletion", &OptionCommentAllowDeletion{})

	// `_group-by` (dodder#374(b) OQ3 ruling): lives on the base blob's
	// OWN envelope metadata, never the outer organize document's --
	// registered here anyway (harmless no-op if it ever appears on an
	// outer document) rather than giving the envelope its own
	// constructor for one extra prototype.
	ocs.AddPrototype("group-by", &OptionCommentGroupBy{})

	return ocs
}

type PrototypeOptionComments map[string]OptionComment

type OptionCommentSet struct {
	prototype      PrototypeOptionComments
	OptionComments []OptionComment
}

func (ocs *OptionCommentSet) GetPrototypeOptionComments() PrototypeOptionComments {
	return ocs.prototype
}

// GetByKey returns the first active OptionComment registered under
// key from OptionComments (the document's ACTIVE settings, not the
// prototype registry), if any, without removing it.
func (ocs *OptionCommentSet) GetByKey(key string) (found OptionComment, ok bool) {
	for _, oc := range ocs.OptionComments {
		ocwk, isKeyed := oc.(OptionCommentWithKey)
		if !isKeyed || ocwk.Key != key {
			continue
		}

		return oc, true
	}

	return found, false
}

// RemoveByKey removes and returns the first active OptionComment
// registered under key from OptionComments (the document's ACTIVE
// settings, not the prototype registry -- GetPrototypeOptionComments
// is unaffected), if any. dodder#374(b) plan §4: patch's `_base` entry
// must be dropped structurally before diffing against the base blob's
// body, which never had one (rendered before `_base` was inserted) --
// the dual of how WriteOrganizeBaseAndActivate ADDS it via
// AddPrototypeAndOption. Confirmed structural, not string-level
// surgery on raw text, per the 2026-07-19 review.
func (ocs *OptionCommentSet) RemoveByKey(key string) (removed OptionComment, found bool) {
	for i, oc := range ocs.OptionComments {
		ocwk, ok := oc.(OptionCommentWithKey)
		if !ok || ocwk.Key != key {
			continue
		}

		removed = oc
		found = true
		ocs.OptionComments = append(
			ocs.OptionComments[:i],
			ocs.OptionComments[i+1:]...,
		)

		return removed, found
	}

	return removed, found
}

func (ocs *OptionCommentSet) AddPrototype(
	key string,
	o OptionComment,
) OptionComment {
	o = OptionCommentWithKey{
		Key:           key,
		OptionComment: o,
	}

	ocs.prototype[key] = o

	return o
}

func (ocs *OptionCommentSet) AddPrototypeAndOption(
	key string,
	o OptionComment,
) OptionComment {
	o = ocs.AddPrototype(key, o)
	ocs.OptionComments = append(ocs.OptionComments, o)
	return o
}

func (ocs *OptionCommentSet) Set(v string) (err error) {
	head, tail, _ := strings.Cut(v, ":")

	oc, ok := ocs.prototype[head]

	if ok {
		// ocs.prototype entries are already OptionCommentWithKey (see
		// AddPrototype below); unwrap before cloning so
		// CloneOptionComment() clones the actual registered comment
		// (e.g. *OptionCommentDryRun) rather than cloning the wrapper
		// itself, which would re-wrap on the next line and produce a
		// double-wrapped OptionCommentWithKey{OptionComment:
		// OptionCommentWithKey{...}}. That double-wrap broke both
		// String() (rendering "key:key:value") and any interface
		// assertion against the inner concrete type (e.g.
		// OptionCommentSettingsField), since Go's embedded-interface
		// promotion only forwards the embedded interface's own declared
		// methods, not the wrapped-again value's extra methods.
		if ocwk, isWrapped := oc.(OptionCommentWithKey); isWrapped {
			oc = ocwk.OptionComment
		}

		oc = oc.CloneOptionComment()
	} else {
		oc = &OptionCommentUnknown{}
	}

	oc = OptionCommentWithKey{
		Key:           head,
		OptionComment: oc,
	}

	if err = oc.Set(tail); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocs.OptionComments = append(
		ocs.OptionComments,
		oc,
	)

	return err
}

// TODO add support for ApplyTo*
type OptionCommentWithKey struct {
	Key string
	OptionComment
}

func (ocf OptionCommentWithKey) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf OptionCommentWithKey) Set(v string) (err error) {
	if err = ocf.OptionComment.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (ocf OptionCommentWithKey) String() string {
	return fmt.Sprintf("%s:%s", ocf.Key, ocf.OptionComment)
}

type optionCommentHide string

func (ocf optionCommentHide) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf optionCommentHide) Set(v string) (err error) {
	return comments.Implement()
}

func (ocf optionCommentHide) String() string {
	return fmt.Sprintf("hide:%s", string(ocf))
}

func (ocf optionCommentHide) ApplyToText(Options, *Assignment) (err error) {
	return err
}

func (ocf optionCommentHide) ApplyToReader(
	Options,
	*reader,
) (err error) {
	return err
}

func (ocf optionCommentHide) ApplyToWriter(
	f Options,
	aw *writer,
) (err error) {
	return err
}

type OptionCommentDryRun struct {
	mad_domain_interfaces.MutableConfigDryRun
}

func (ocf *OptionCommentDryRun) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf *OptionCommentDryRun) Set(v string) (err error) {
	var boolValue values.Bool

	if err = boolValue.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocf.SetDryRun(boolValue.Bool())

	return err
}

func (ocf *OptionCommentDryRun) String() string {
	return fmt.Sprintf("%t", ocf.IsDryRun())
}

func (ocf *OptionCommentDryRun) IsSettingsField() bool {
	return true
}

// OptionCommentBaseDigest is `- _base=@<digest>` (dodder#374(b),
// cutting-garden RFC 0015 / hyphence RFC 0002's id-less, digest-valued
// FieldRHS). Required on every organize document; see the plan's §8 for
// the missing/undereferenceable error paths, which live in the
// read/apply flow, not here -- this type is parsing only.
type OptionCommentBaseDigest struct {
	Id markl.Id
}

func (ocf *OptionCommentBaseDigest) CloneOptionComment() OptionComment {
	clone := *ocf
	return &clone
}

func (ocf *OptionCommentBaseDigest) Set(v string) (err error) {
	if err = ocf.Id.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (ocf *OptionCommentBaseDigest) String() string {
	// markl.Id.String() omits the leading "@" (it accepts "@..." on Set
	// but doesn't echo it back on String) -- re-add it so WriteTo's
	// output is the RFC-0002-shaped "- _base=@<digest>", not
	// "- _base=<digest>".
	return "@" + ocf.Id.String()
}

func (ocf *OptionCommentBaseDigest) IsSettingsField() bool {
	return true
}

// OptionCommentAllowDeletion is `- _allow-deletion=true` (dodder#374(b)),
// the first of the deletion gates -- see the plan's §7 for the other
// three (deletion-set computation, post-editor confirmation, and
// commit-directly's additional CLI flag), none of which live here.
type OptionCommentAllowDeletion struct {
	Value bool
}

func (ocf *OptionCommentAllowDeletion) CloneOptionComment() OptionComment {
	clone := *ocf
	return &clone
}

func (ocf *OptionCommentAllowDeletion) Set(v string) (err error) {
	var boolValue values.Bool

	if err = boolValue.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocf.Value = boolValue.Bool()

	return err
}

func (ocf *OptionCommentAllowDeletion) String() string {
	return fmt.Sprintf("%t", ocf.Value)
}

func (ocf *OptionCommentAllowDeletion) IsSettingsField() bool {
	return true
}

// OptionCommentGroupBy is `- _group-by="tag1,tag2"` (dodder#374(b), OQ3
// ruling): the base blob's own envelope metadata records the -group-by
// value(s) used at generation (absent = ungrouped), so grouped-detection
// (plan §5) reads it from the base blob's own structure rather than
// inferring it from the patch. Value quoted and comma-joined (no spaces)
// to preserve -group-by's order in one field -- RFC 0001's metadata
// lines are order-independent across lines, so an ordered list can't be
// spread across repeated `_group-by=...` lines.
type OptionCommentGroupBy struct {
	Value string
}

func (ocf *OptionCommentGroupBy) CloneOptionComment() OptionComment {
	clone := *ocf
	return &clone
}

func (ocf *OptionCommentGroupBy) Set(v string) (err error) {
	v = strings.TrimPrefix(v, `"`)
	v = strings.TrimSuffix(v, `"`)
	ocf.Value = v

	return err
}

func (ocf *OptionCommentGroupBy) String() string {
	return fmt.Sprintf("%q", ocf.Value)
}

func (ocf *OptionCommentGroupBy) IsSettingsField() bool {
	return true
}

type OptionCommentUnknown struct {
	Value string
}

func (ocf OptionCommentUnknown) CloneOptionComment() OptionComment {
	return &OptionCommentUnknown{Value: ocf.Value}
}

func (ocf *OptionCommentUnknown) Set(v string) (err error) {
	ocf.Value = v
	return err
}

func (ocf OptionCommentUnknown) String() string {
	return ocf.Value
}

type OptionCommentBooleanFlag struct {
	Value   *bool
	Comment string
}

func (ocf OptionCommentBooleanFlag) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf OptionCommentBooleanFlag) Set(v string) (err error) {
	head, tail, _ := strings.Cut(v, " ")

	var b values.Bool

	if err = b.Set(head); err != nil {
		err = errors.Wrap(err)
		return err
	}

	*ocf.Value = b.Bool()

	ocf.Comment = tail

	return err
}

func (ocf OptionCommentBooleanFlag) String() string {
	if ocf.Comment != "" {
		return fmt.Sprintf("%t %s", *ocf.Value, ocf.Comment)
	} else {
		return fmt.Sprintf("%t", *ocf.Value)
	}
}
