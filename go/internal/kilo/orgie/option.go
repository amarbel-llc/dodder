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

	// `base` (data plane) / `allow-deletion` (operational plane, RFC
	// 0015's merged two-plane revision) -- both registered
	// unconditionally, unlike `dry-run` (which only becomes a
	// registered prototype when the CLI's `-dry-run` flag is active,
	// ApplyToOrganizeOptions). `_base` is required on every organize
	// document and `%:allow-deletion` must always be settable by hand --
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

// AddPrototypeAndDirectiveOption is AddPrototypeAndOption's operational-
// plane sibling: registers o as BOTH the prototype for key and an
// immediately-active Setting, marked IsDirective so it renders with
// the "%:key = value" spelling (RFC 0015) rather than the pre-RFC-0015
// "% key:value" spelling AddPrototypeAndOption still produces for
// not-yet-migrated legacy comments. For directives whose generation-
// time activation doesn't come from parsing text -- e.g. SettingDryRun
// reads its display value dynamically from a live config pointer at
// String() time, not from a parsed value string -- so setDirective's
// parse-driven construction doesn't apply; this mirrors
// AddPrototypeAndOption's exact no-clone semantics (the caller's
// freshly-constructed Setting is used directly, not cloned), just with
// IsDirective set.
func (ocs *SettingSet) AddPrototypeAndDirectiveOption(
	key string,
	o Setting,
) Setting {
	wrapped := SettingWithKey{
		Key:         key,
		Setting:     o,
		IsDirective: true,
	}

	ocs.prototype[key] = wrapped
	ocs.Settings = append(ocs.Settings, wrapped)

	return wrapped
}

// RegisterNamespaced registers a driving-command directive prototype
// under the "<namespace>/<name>" key convention -- cutting-garden RFC
// 0015 (merged): a namespaced `%:<command>/<name>` directive (e.g.
// `%:checkin/delete = true`) routes to the driving command, external
// to orgie's own harness-level directives. Reuses the SAME flat
// SettingSet.prototype map AddPrototype already populates for bare
// harness directives (dry-run, allow-deletion) and data-plane fields
// (base, group-by) -- no separate registry type. RFC 0015's "not
// resolved by orgie itself" is satisfied entirely by WHO calls this
// (the driving command, e.g. checkin.go, before running organize -- not
// orgie itself), not by WHERE the prototype is stored: setDirective's
// lookup (option.go) doesn't distinguish a namespaced key from a bare
// one, it just resolves whatever string the parsed directive name is.
func (ocs *SettingSet) RegisterNamespaced(
	namespace, name string,
	o Setting,
) Setting {
	return ocs.AddPrototype(namespace+"/"+name, o)
}

// RegisterNamespacedAndActivate is RegisterNamespaced's immediately-
// active sibling, mirroring AddPrototypeAndDirectiveOption's register-
// and-activate shape with the namespace/name split for call-site
// readability -- checkin.go's real call site for its "delete" flag,
// which needs both a registered prototype (so a user-edited
// `%:checkin/delete = true` line resolves on re-read) and immediate
// generation-time activation (so it's shown in the buffer).
func (ocs *SettingSet) RegisterNamespacedAndActivate(
	namespace, name string,
	o Setting,
) Setting {
	return ocs.AddPrototypeAndDirectiveOption(namespace+"/"+name, o)
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

// SetDirective resolves a `%:`-directive's text (everything after the
// colon, e.g. "dry-run = true" or "checkin/delete = true") against the
// SAME prototype map Set() uses for `%`-comments and `- _key = value`
// data-plane fields -- cutting-garden RFC 0015 (merged): "not resolved
// by orgie itself" for a namespaced directive is satisfied by WHO
// registers the prototype (the driving command, via RegisterNamespaced),
// not WHERE it's stored, so no separate registry is needed. Presence-
// only booleans (bare "%:dry-run", no "=") default to "true". An
// unrecognized name is an error (ErrUnrecognizedDirective), not a silent
// SettingUnknown fallback: directives are behavior-bearing by
// construction, so silently ignoring one would silently skip behavior
// the user asked for -- prose (AddInertProse) is the only operational-
// plane shape that's silently tolerant when unrecognized. The ONE
// exception is setDirectiveLegacyAlias (below), used ONLY for the
// pre-RFC-0015 `% key:value` back-compat shim, which predates the
// error-on-unrecognized rule and must keep its original tolerance.
func (ocs *SettingSet) SetDirective(directiveText string) (err error) {
	return ocs.setDirective(directiveText, false)
}

// setDirectiveLegacyAlias resolves the pre-RFC-0015 `% key:value`
// comment spelling (e.g. `% dry-run:true`, reformatted by
// Metadata.readOperationalPlaneLine into "key = value" before reaching
// here) with the SAME tolerance the original SettingSet.Set always had:
// an unrecognized name silently falls back to an inert placeholder
// instead of erroring (dodder's existing "a settings field this run
// doesn't recognize can't be distinguished from one it simply isn't
// active for" contract -- e.g. `% dry-run:true` read in a context
// without -dry-run on the CLI, so "dry-run" was never registered, is a
// legitimate no-op, pinned at the bats level by
// organize_dry_run_legacy_comment_alias_still_accepted). This is
// deliberately NOT the same tolerance as a genuine new `%:` directive
// (SetDirective, strict per RFC 0015) -- the legacy alias exists purely
// to tolerate old documents gracefully, a dodder-local migration
// concern the RFC doesn't govern.
func (ocs *SettingSet) setDirectiveLegacyAlias(directiveText string) (err error) {
	return ocs.setDirective(directiveText, true)
}

func (ocs *SettingSet) setDirective(
	directiveText string,
	tolerateUnrecognized bool,
) (err error) {
	name, value, hasValue := strings.Cut(directiveText, "=")
	name = strings.TrimSpace(name)

	if hasValue {
		value = strings.TrimSpace(value)
	} else {
		value = "true"
	}

	oc, ok := ocs.prototype[name]

	if ok {
		if ocwk, isWrapped := oc.(SettingWithKey); isWrapped {
			oc = ocwk.Setting
		}

		oc = oc.CloneSetting()
	} else if tolerateUnrecognized {
		oc = &SettingUnknown{}
	} else {
		err = errors.Wrap(ErrUnrecognizedDirective{Name: name})
		return err
	}

	oc = SettingWithKey{
		Key:         name,
		Setting:     oc,
		IsDirective: true,
	}

	if err = oc.Set(value); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocs.Settings = append(ocs.Settings, oc)

	return err
}

// AddInertProse records a bare `% <text>` line (cutting-garden RFC 0015,
// merged) -- inert by construction, never parsed for structure, kept
// only for round-trip fidelity. Deliberately stored UNWRAPPED (not
// SettingWithKey, which has no meaningful "key" for free text) so
// writeTo can distinguish "has a key -> directive" from "no key ->
// prose" by type shape alone; SettingUnknown.String() returns its
// Value verbatim, so "%% %s" renders exactly "% <text>" back out.
func (ocs *SettingSet) AddInertProse(text string) {
	ocs.Settings = append(ocs.Settings, &SettingUnknown{Value: text})
}

// TODO add support for ApplyTo*
type SettingWithKey struct {
	Key string
	Setting

	// IsDirective marks this INSTANCE (not Setting type -- the same
	// concrete type, e.g. SettingBooleanFlag, is used both by genuine
	// %: directives and by not-yet-migrated legacy %-comments like
	// checkin's "delete" flag) as constructed via setDirective (RFC
	// 0015's `%:name = value` syntax or its `% dry-run:true` legacy
	// alias), as opposed to the pre-RFC-0015 Set() path (data-plane
	// `- _key = value` fields and legacy `% key:value` comments not yet
	// migrated to the new syntax, e.g. checkin's "delete" flag until
	// piece 4 migrates it). Distinguishes writeTo's two operational-
	// plane write forms ("%:key = value" vs "% key:value") per
	// instance. Zero value (false) preserves the pre-existing "%
	// key:value" spelling for every construction site that doesn't set
	// it explicitly.
	IsDirective bool
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

// SettingDryRun does NOT implement SettingAsField (unlike SettingBaseDigest/
// SettingGroupBy below): cutting-garden RFC 0015's merged two-plane
// revision (ruled 2026-07-28) reclassifies `_dry-run` from the data
// plane (`- _dry-run = true`) to an OPERATIONAL-plane directive
// (`%:dry-run = true`) -- it configures how the apply behaves, not
// what the document is, so it belongs on the plane that's stripped
// from the base blob. Activated via AddPrototypeAndDirectiveOption
// (organize_options.go's ApplyToOrganizeOptions), not
// AddPrototypeAndOption, so its generation-time activation is also
// marked IsDirective and renders with the "%:" spelling.

// SettingBaseDigest is `- _base = @<digest>` (dodder#374(b),
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
	// output is the RFC-0002-shaped "- _base = @<digest>", not
	// "- _base = <digest>". Set (above) strips it back off before
	// delegating to markl.Id.Set, which treats a leading "@" as a
	// (rejected, empty) purpose delimiter, not a bare-digest marker.
	return "@" + ocf.Id.String()
}

func (ocf *SettingBaseDigest) IsSettingsField() bool {
	return true
}

// SettingAllowDeletion is `%:allow-deletion = true` (dodder#374(b);
// re-spelled from the data-plane `- _allow-deletion = true` by
// cutting-garden RFC 0015's merged two-plane revision, ruled
// 2026-07-28 -- it's a mutation-permitting directive, not a document
// field, so it belongs on the operational plane, stripped from the
// base blob). Parsing only -- kept for RFC 0015 cross-substrate
// document portability (`%:allow-deletion = true` must round-trip
// without erroring), but dodder does not enforce it: the plan's §7
// gate exists to guard true substrate deletion (an object ceasing to
// exist), and dodder has no such operation anywhere -- organize's
// tag-clearing (changes.go, three_way.go) only ever mutates an
// existing object's tags. See the plan's §7 (2026-07-19 ruling) for
// the full rationale, including why generalizing to "any tag-clear
// that fully untags an object" was considered and rejected.
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

// No IsSettingsField() -- see the type's own doc comment: reclassified
// to the operational plane by RFC 0015's merged revision. Activated
// only by reading a user-authored `%:allow-deletion = true` line
// (setDirective, which marks IsDirective), never auto-activated at
// generation the way SettingDryRun sometimes is, so no
// AddPrototypeAndDirectiveOption call site is needed for this type.

// SettingGroupBy is `- _group-by = "tag1,tag2"` (dodder#374(b), OQ3
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
