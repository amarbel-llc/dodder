package env_repo

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// wrapConfigSeedDecodeError converts a hyphence/markl decode error from
// the config-seed file into a user-facing ErrLegacyConfigSeedKey when
// the underlying cause is the pre-madder-v0.3.16 combined-HRP wire
// form. The returned error implements interfaces.ErrorRetryable, so the
// dewey context runner will route it through Recover() before
// surfacing it. Other errors are returned with the standard errors.Wrap
// stack.
func wrapConfigSeedDecodeError(
	err error,
	envUI env_ui.Env,
	path string,
) error {
	if err == nil {
		return nil
	}

	var legacy markl.ErrLegacyCombinedHRPWireForm
	if errors.As(err, &legacy) {
		return ErrLegacyConfigSeedKey{
			envUI:    envUI,
			Path:     path,
			Purpose:  legacy.Purpose,
			FormatId: legacy.FormatId,
			Inner:    err,
		}
	}

	return errors.Wrap(err)
}

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrNotInDodderDir struct {
	Expected string
}

func (err ErrNotInDodderDir) Error() string {
	if err.Expected == "" {
		return "not in a dodder directory."
	} else {
		return fmt.Sprintf("not in a dodder directory. Looking for %s", err.Expected)
	}
}

func (err ErrNotInDodderDir) ShouldShowStackTrace() bool {
	return false
}

func (err ErrNotInDodderDir) Is(target error) (ok bool) {
	_, ok = target.(ErrNotInDodderDir)
	return ok
}

func (err ErrNotInDodderDir) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

// ErrLegacyRepoLayout is returned instead of ErrNotInDodderDir when the
// expected nested repos/<name>/config-seed is missing but a legacy
// flat-layout config-seed (pre-FDR-0019, no repos/ nesting) exists at the
// same scope root. FDR-0019 deliberately does not recognize a legacy tree
// in place (#363: explicit-migrate-only, not read-in-place) -- this error
// exists so that decision surfaces as a specific, actionable message
// naming `migrate-repo-layout` instead of the generic "not in a dodder
// directory" ErrNotInDodderDir, for both human users and agents.
type ErrLegacyRepoLayout struct {
	// LegacyPath is the pre-FDR-0019 flat-layout config-seed found at the
	// scope root (e.g. <scope>/local/share/config-seed).
	LegacyPath string
	// ScopeRoot is LegacyPath's directory -- the legacy tree's root,
	// suitable as migrate-repo-layout's -source.
	ScopeRoot string
	// Name is the repo name that was being resolved (e.g. "default").
	Name string
}

func (err ErrLegacyRepoLayout) Error() string {
	return fmt.Sprintf(
		"%s is a legacy (pre-FDR-0019) repo layout, not readable in place; migrate it first",
		err.ScopeRoot,
	)
}

func (err ErrLegacyRepoLayout) ShouldShowStackTrace() bool {
	return false
}

func (err ErrLegacyRepoLayout) Is(target error) (ok bool) {
	_, ok = target.(ErrLegacyRepoLayout)
	return ok
}

func (err ErrLegacyRepoLayout) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrLegacyRepoLayout) GetErrorCause() []string {
	return []string{
		fmt.Sprintf(
			"%s holds a legacy config-seed directly at the scope root, from before FDR-0019's `repos/<name>/` nesting.",
			err.ScopeRoot,
		),
		"The current binary only reads repos nested under repos/<name>/ -- it does not recognize a legacy flat tree in place (a deliberate scope decision, not a bug).",
	}
}

func (err ErrLegacyRepoLayout) GetErrorRecovery() []string {
	return []string{
		fmt.Sprintf(
			"Migrate it: `dodder migrate-repo-layout -source %q -dest <new-path> -name %s` (never modifies -source).",
			err.ScopeRoot,
			err.Name,
		),
		"Then point at the migrated tree (e.g. `-dir-dodder <new-path>`) and verify with a read-only command (`dodder show`) before treating the old tree as disposable.",
	}
}

// ErrLegacyConfigSeedKey is returned when the `private-key` markl-id in
// a config-seed file is in the pre-madder-v0.3.16 combined-HRP wire
// form. It implements interfaces.ErrorRetryable: the dewey context
// runner invokes Recover, which prints a Helpful cause/recovery block,
// asks the user whether to migrate in place, and on confirmation calls
// MigrateLegacyCombinedHRPConfigSeed before retrying the original
// context body.
type ErrLegacyConfigSeedKey struct {
	envUI    env_ui.Env
	Path     string
	Purpose  string
	FormatId string
	Inner    error
}

var _ interfaces.ErrorRetryable = ErrLegacyConfigSeedKey{}

func (err ErrLegacyConfigSeedKey) Error() string {
	return fmt.Sprintf(
		"config-seed at %s holds a %s private-key in the legacy combined-HRP wire form",
		err.Path,
		err.FormatId,
	)
}

func (err ErrLegacyConfigSeedKey) ShouldShowStackTrace() bool {
	return false
}

func (err ErrLegacyConfigSeedKey) Is(target error) (ok bool) {
	_, ok = target.(ErrLegacyConfigSeedKey)
	return ok
}

func (err ErrLegacyConfigSeedKey) Unwrap() error {
	return err.Inner
}

func (err ErrLegacyConfigSeedKey) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrLegacyConfigSeedKey) GetErrorCause() []string {
	return []string{
		fmt.Sprintf(
			"The %s private-key in %s was written by pre-madder v0.3.16, which used the combined-HRP wire form (purpose@format as a single blech32 HRP).",
			err.FormatId,
			err.Path,
		),
		"The current canonical form (RFC 0002 §3.3) binds the blech32 checksum to the format only and prepends the purpose textually after encoding.",
		"The underlying key bytes are intact; only the trailing 6-char blech32 checksum is wrong. See madder#167 for the wire-format drift story.",
	}
}

func (err ErrLegacyConfigSeedKey) GetErrorRecovery() []string {
	return []string{
		"The key can be re-encoded in place; no re-signing or key rotation is required.",
		fmt.Sprintf("Equivalent CLI invocation: `der migrate-config-seed-key -path %q`.", err.Path),
	}
}

func (err ErrLegacyConfigSeedKey) Recover(
	ctx interfaces.ActiveContext,
	retry interfaces.FuncRetry,
	abort interfaces.FuncRetryAborted,
) {
	errors.PrintHelpful(err.envUI.GetErr(), err)

	prompt := fmt.Sprintf(
		"migrate %s in place (writes .bak.<unix-seconds> sibling first)?",
		err.Path,
	)

	if !err.envUI.Confirm(prompt, "") {
		abort(errors.Errorf("declined to migrate config-seed key"))
		return
	}

	result, mErr := MigrateLegacyCombinedHRPConfigSeed(err.Path, true, false)
	if mErr != nil {
		abort(mErr)
		return
	}

	if result.BackupPath != "" {
		err.envUI.GetErr().Printf("wrote backup: %s", result.BackupPath)
	}
	err.envUI.GetErr().Printf("rewrote %s", err.Path)

	retry()
}
