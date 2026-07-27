package orgie

import (
	"fmt"
	"strconv"
	"strings"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"

	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/values"
)

type Setting interface {
	CloneSetting() Setting
	interfaces.StringerSetter
}

type SettingWithApply interface {
	Setting
	ApplyToText(Options, *Assignment) error
	ApplyToReader(Options, *reader) error
	ApplyToWriter(Options, *writer) error
}

// SettingAsField marks a Setting as a `_`-reserved document settings
// field (dodder#374, cutting-garden RFC 0015) rather than a `%` comment.
// Implementing this is the only step needed to migrate a Setting's
// write-side spelling -- Metadata.WriteTo checks it generically, matching
// ReadFrom's already-generic prototype-keyed parsing, so no new
// type-switch branch is needed per migrated setting.
type SettingAsField interface {
	Setting
	IsSettingsField() bool
}

// isSettingsField reports whether a single-wrapped SettingWithKey (from
// SettingSet's prototype registry or Settings slice) should use the `-
// _key=value` settings-field spelling rather than `% key:value`. Shared
// by Metadata.ReadFrom (deciding whether a `- _key=value` line is a
// recognized settings field or falls through to ordinary tag parsing)
// and Metadata.WriteTo (deciding how to emit an already-registered
// Setting), so the two directions can't drift.
func isSettingsField(o Setting) bool {
	ocwk, ok := o.(SettingWithKey)
	if !ok {
		return false
	}

	sf, ok := ocwk.Setting.(SettingAsField)

	return ok && sf.IsSettingsField()
}

// TODO add config to automatically add dry run if necessary
func MakeSettingSet(
	elements map[string]Setting,
	options ...Setting,
) SettingSet {
	ocs := SettingSet{
		prototype: make(PrototypeSettings),
		Settings:  options,
	}

	if elements != nil {
		for k, el := range elements {
			ocs.AddPrototype(k, el)
		}
	}

	ocs.AddPrototype("hide", settingHide(""))
	ocs.AddPrototype("", settingHide(""))

	// `_base`/`_allow-deletion` (dodder#374(b), cutting-garden RFC 0015):
	// registered unconditionally, unlike `_dry-run` (which only becomes a
	// registered prototype when the CLI's `-dry-run` flag is active,
	// ApplyToOrganizeOptions). `_base` is required on every organize
	// document and `_allow-deletion` must always be settable by hand --
	// neither mirrors ambient CLI/config state the way dry-run does.
	ocs.AddPrototype("base", &SettingBaseDigest{})
	ocs.AddPrototype("allow-deletion", &SettingAllowDeletion{})

	// `_group-by` (dodder#374(b) OQ3 ruling): lives on the base blob's
	// OWN envelope metadata, never the outer organize document's --
	// registered here anyway (harmless no-op if it ever appears on an
	// outer document) rather than giving the envelope its own
	// constructor for one extra prototype.
	ocs.AddPrototype("group-by", &SettingGroupBy{})

	return ocs
}

type PrototypeSettings map[string]Setting

type SettingSet struct {
	prototype PrototypeSettings
	Settings  []Setting
}

func (ocs *SettingSet) GetPrototypeSettings() PrototypeSettings {
	return ocs.prototype
}

// GetByKey returns the first active Setting registered under key from
// Settings (the document's ACTIVE settings, not the prototype registry),
// if any, without removing it.
func (ocs *SettingSet) GetByKey(key string) (found Setting, ok bool) {
	for _, oc := range ocs.Settings {
		ocwk, isKeyed := oc.(SettingWithKey)
		if !isKeyed || ocwk.Key != key {
			continue
		}

		return oc, true
	}

	return found, false
}

// RemoveByKey removes and returns the first active Setting registered
// under key from Settings (the document's ACTIVE settings, not the
// prototype registry -- GetPrototypeSettings is unaffected), if any.
// dodder#374(b) plan §4: patch's `_base` entry
// must be dropped structurally before diffing against the base blob's
// body, which never had one (rendered before `_base` was inserted) --
// the dual of how WriteOrganizeBaseAndActivate ADDS it via
// AddPrototypeAndOption. Confirmed structural, not string-level
// surgery on raw text, per the 2026-07-19 review.
func (ocs *SettingSet) RemoveByKey(key string) (removed Setting, found bool) {
	for i, oc := range ocs.Settings {
		ocwk, ok := oc.(SettingWithKey)
		if !ok || ocwk.Key != key {
			continue
		}

		removed = oc
		found = true
		ocs.Settings = append(
			ocs.Settings[:i],
			ocs.Settings[i+1:]...,
		)

		return removed, found
	}

	return removed, found
}

func (ocs *SettingSet) AddPrototype(
	key string,
	o Setting,
) Setting {
	o = SettingWithKey{
		Key:     key,
		Setting: o,
	}

	ocs.prototype[key] = o

	return o
}

func (ocs *SettingSet) AddPrototypeAndOption(
	key string,
	o Setting,
) Setting {
	o = ocs.AddPrototype(key, o)
	ocs.Settings = append(ocs.Settings, o)
	return o
}

func (ocs *SettingSet) Set(v string) (err error) {
	head, tail, _ := strings.Cut(v, ":")

	oc, ok := ocs.prototype[head]

	if ok {
		// ocs.prototype entries are already SettingWithKey (see
		// AddPrototype below); unwrap before cloning so
		// CloneSetting() clones the actual registered comment
		// (e.g. *SettingDryRun) rather than cloning the wrapper
		// itself, which would re-wrap on the next line and produce a
		// double-wrapped SettingWithKey{Setting:
		// SettingWithKey{...}}. That double-wrap broke both
		// String() (rendering "key:key:value") and any interface
		// assertion against the inner concrete type (e.g.
		// SettingAsField), since Go's embedded-interface
		// promotion only forwards the embedded interface's own declared
		// methods, not the wrapped-again value's extra methods.
		if ocwk, isWrapped := oc.(SettingWithKey); isWrapped {
			oc = ocwk.Setting
		}

		oc = oc.CloneSetting()
	} else {
		oc = &SettingUnknown{}
	}

	oc = SettingWithKey{
		Key:     head,
		Setting: oc,
	}

	if err = oc.Set(tail); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocs.Settings = append(
		ocs.Settings,
		oc,
	)

	return err
}

// TODO add support for ApplyTo*
type SettingWithKey struct {
	Key string
	Setting
}

func (ocf SettingWithKey) CloneSetting() Setting {
	return ocf
}

func (ocf SettingWithKey) Set(v string) (err error) {
	if err = ocf.Setting.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (ocf SettingWithKey) String() string {
	return fmt.Sprintf("%s:%s", ocf.Key, ocf.Setting)
}

type settingHide string

func (ocf settingHide) CloneSetting() Setting {
	return ocf
}

func (ocf settingHide) Set(v string) (err error) {
	return comments.Implement()
}

func (ocf settingHide) String() string {
	return fmt.Sprintf("hide:%s", string(ocf))
}

func (ocf settingHide) ApplyToText(Options, *Assignment) (err error) {
	return err
}

func (ocf settingHide) ApplyToReader(
	Options,
	*reader,
) (err error) {
	return err
}

func (ocf settingHide) ApplyToWriter(
	f Options,
	aw *writer,
) (err error) {
	return err
}

type SettingDryRun struct {
	mad_domain_interfaces.MutableConfigDryRun
}

func (ocf *SettingDryRun) CloneSetting() Setting {
	return ocf
}

func (ocf *SettingDryRun) Set(v string) (err error) {
	var boolValue values.Bool

	if err = boolValue.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocf.SetDryRun(boolValue.Bool())

	return err
}

func (ocf *SettingDryRun) String() string {
	return fmt.Sprintf("%t", ocf.IsDryRun())
}

func (ocf *SettingDryRun) IsSettingsField() bool {
	return true
}

// SettingBaseDigest is `- _base=@<digest>` (dodder#374(b),
// cutting-garden RFC 0015 / hyphence RFC 0002's id-less, digest-valued
// FieldRHS). Required on every organize document; see the plan's §8 for
// the missing/undereferenceable error paths, which live in the
// read/apply flow, not here -- this type is parsing only.
type SettingBaseDigest struct {
	Id markl.Id
}

func (ocf *SettingBaseDigest) CloneSetting() Setting {
	clone := *ocf
	return &clone
}

func (ocf *SettingBaseDigest) Set(v string) (err error) {
	// Strip the leading "@" before delegating -- it's this type's own
	// display convention (String() below), not something markl.Id.Set
	// understands. markl.Id.Set's wire form is `[purpose@]<digest>`
	// (splits on the first "@"), so passing "@<digest>" through
	// unstripped is read as an EMPTY PURPOSE, which the library rejects
	// ("invalid bare purpose"). Confirmed against a real failure this
	// caused in WriteOrganizeBaseAndActivate (organize_base.go) after a
	// markl dependency bump tightened this validation.
	v = strings.TrimPrefix(v, "@")

	if err = ocf.Id.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (ocf *SettingBaseDigest) String() string {
	// markl.Id.String() never includes "@" -- re-add it so WriteTo's
	// output is the RFC-0002-shaped "- _base=@<digest>", not
	// "- _base=<digest>". Set (above) strips it back off before
	// delegating to markl.Id.Set, which treats a leading "@" as a
	// (rejected, empty) purpose delimiter, not a bare-digest marker.
	return "@" + ocf.Id.String()
}

func (ocf *SettingBaseDigest) IsSettingsField() bool {
	return true
}

// SettingAllowDeletion is `- _allow-deletion=true` (dodder#374(b)).
// Parsing only -- kept for RFC 0015 cross-substrate document
// portability (`- _allow-deletion=true` must round-trip without
// erroring), but dodder does not enforce it: the plan's §7 gate exists
// to guard true substrate deletion (an object ceasing to exist), and
// dodder has no such operation anywhere -- organize's tag-clearing
// (changes.go, three_way.go) only ever mutates an existing object's
// tags. See the plan's §7 (2026-07-19 ruling) for the full rationale,
// including why generalizing to "any tag-clear that fully untags an
// object" was considered and rejected.
type SettingAllowDeletion struct {
	Value bool
}

func (ocf *SettingAllowDeletion) CloneSetting() Setting {
	clone := *ocf
	return &clone
}

func (ocf *SettingAllowDeletion) Set(v string) (err error) {
	var boolValue values.Bool

	if err = boolValue.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocf.Value = boolValue.Bool()

	return err
}

func (ocf *SettingAllowDeletion) String() string {
	return fmt.Sprintf("%t", ocf.Value)
}

func (ocf *SettingAllowDeletion) IsSettingsField() bool {
	return true
}

// SettingGroupBy is `- _group-by="tag1,tag2"` (dodder#374(b), OQ3
// ruling): the base blob's own envelope metadata records the -group-by
// value(s) used at generation (absent = ungrouped), so grouped-detection
// (plan §5) reads it from the base blob's own structure rather than
// inferring it from the patch. Value quoted and comma-joined (no spaces)
// to preserve -group-by's order in one field -- RFC 0001's metadata
// lines are order-independent across lines, so an ordered list can't be
// spread across repeated `_group-by=...` lines.
type SettingGroupBy struct {
	Value string
}

func (ocf *SettingGroupBy) CloneSetting() Setting {
	clone := *ocf
	return &clone
}

func (ocf *SettingGroupBy) Set(v string) (err error) {
	// strconv.Unquote (not a manual TrimPrefix/TrimSuffix) so this is the
	// true inverse of String()'s %q below -- a value containing a literal
	// `"` or `\` round-trips correctly, matching the established pattern
	// for quoted hyphence field values (object_metadata_fmt_hyphence's
	// text_parser2.go).
	if unquoted, unquoteErr := strconv.Unquote(v); unquoteErr == nil {
		v = unquoted
	}

	ocf.Value = v

	return err
}

func (ocf *SettingGroupBy) String() string {
	return fmt.Sprintf("%q", ocf.Value)
}

func (ocf *SettingGroupBy) IsSettingsField() bool {
	return true
}

type SettingUnknown struct {
	Value string
}

func (ocf SettingUnknown) CloneSetting() Setting {
	return &SettingUnknown{Value: ocf.Value}
}

func (ocf *SettingUnknown) Set(v string) (err error) {
	ocf.Value = v
	return err
}

func (ocf SettingUnknown) String() string {
	return ocf.Value
}

type SettingBooleanFlag struct {
	Value   *bool
	Comment string
}

func (ocf SettingBooleanFlag) CloneSetting() Setting {
	return ocf
}

func (ocf SettingBooleanFlag) Set(v string) (err error) {
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

func (ocf SettingBooleanFlag) String() string {
	if ocf.Comment != "" {
		return fmt.Sprintf("%t %s", *ocf.Value, ocf.Comment)
	} else {
		return fmt.Sprintf("%t", *ocf.Value)
	}
}
