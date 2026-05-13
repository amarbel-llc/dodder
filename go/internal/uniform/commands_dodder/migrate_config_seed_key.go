package commands_dodder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func init() {
	utility.AddCmd("migrate-config-seed-key", &MigrateConfigSeedKey{
		backup: true,
	})
}

type MigrateConfigSeedKey struct {
	path   string
	backup bool
}

var (
	_ interfaces.CommandComponentWriter = (*MigrateConfigSeedKey)(nil)
	_ command.CommandWithArgs           = (*MigrateConfigSeedKey)(nil)
)

func (cmd MigrateConfigSeedKey) GetDescription() command.Description {
	return command.Description{
		Short: "re-encode config-seed private key in canonical split-HRP form",
		Long: "Rewrites the `private-key` markl-id in dodder's config-seed " +
			"from the legacy combined-HRP wire form (pre-madder v0.3.16) to " +
			"the canonical split-HRP wire form. The underlying key bytes are " +
			"unchanged; only the checksum encoding is fixed. Run once if " +
			"dodder fails to load this repo with an " +
			"ErrLegacyCombinedHRPWireForm on the dodder-repo-private_key " +
			"purpose. See madder#167 and madder#170 for the migration story.",
	}
}

func (cmd *MigrateConfigSeedKey) GetArgs() []command.ArgGroup { return nil }

func (cmd *MigrateConfigSeedKey) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(
		&cmd.path,
		"path",
		"",
		"explicit path to config-seed (default: $XDG_DATA_HOME/dodder/config-seed)",
	)
	flagSet.BoolVar(
		&cmd.backup,
		"backup",
		true,
		"write a .bak.<unix-seconds> sibling before overwriting",
	)
}

var reConfigSeedPrivateKey = regexp.MustCompile(
	`(?m)^(private-key\s*=\s*['"])([^'"]+)(['"])`,
)

func (cmd MigrateConfigSeedKey) Run(req command.Request) {
	req.AssertNoMoreArgs()

	dryRun := repo_config_cli.FromAny(req.Utility.GetConfigAny()).IsDryRun()

	path := cmd.path
	if path == "" {
		resolved, err := defaultConfigSeedPath()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}
		path = resolved
	}

	original, err := os.ReadFile(path)
	if err != nil {
		errors.ContextCancelWithError(req, errors.Wrapf(err, "reading %s", path))
		return
	}

	match := reConfigSeedPrivateKey.FindSubmatchIndex(original)
	if match == nil {
		errors.ContextCancelWithErrorf(
			req,
			"no private-key field found in %s",
			path,
		)
		return
	}

	valueStart, valueEnd := match[4], match[5]
	legacyWire := string(original[valueStart:valueEnd])

	var probe markl.Id
	decodeErr := probe.UnmarshalText([]byte(legacyWire))
	if decodeErr == nil {
		ui.Out().Printf(
			"%s: private-key already in canonical split-HRP wire form, nothing to do",
			path,
		)
		return
	}

	var legacy markl.ErrLegacyCombinedHRPWireForm
	if !errors.As(decodeErr, &legacy) {
		errors.ContextCancelWithError(
			req,
			errors.Wrapf(decodeErr, "decoding private-key in %s", path),
		)
		return
	}

	var canonical markl.Id
	if err := canonical.SetPurposeId(legacy.Purpose); err != nil {
		errors.ContextCancelWithError(
			req,
			errors.Wrapf(err, "setting purpose %q", legacy.Purpose),
		)
		return
	}

	if err := canonical.SetMarklId(legacy.FormatId, legacy.Data); err != nil {
		errors.ContextCancelWithError(
			req,
			errors.Wrapf(
				err,
				"setting format=%q data=%d bytes",
				legacy.FormatId,
				len(legacy.Data),
			),
		)
		return
	}

	canonicalBytes, err := canonical.MarshalText()
	if err != nil {
		errors.ContextCancelWithError(
			req,
			errors.Wrapf(err, "re-marshaling private-key"),
		)
		return
	}

	canonicalWire := string(canonicalBytes)

	if canonicalWire == legacyWire {
		errors.ContextCancelWithErrorf(
			req,
			"re-encode produced identical wire form; refusing to write (purpose=%q, format=%q)",
			legacy.Purpose,
			legacy.FormatId,
		)
		return
	}

	ui.Out().Printf("legacy combined-HRP wire form detected at %s", path)
	ui.Out().Printf("  purpose: %s", legacy.Purpose)
	ui.Out().Printf("  format:  %s", legacy.FormatId)
	ui.Out().Printf("  data:    %d bytes (preserved)", len(legacy.Data))
	ui.Out().Printf("  before:  %s", legacyWire)
	ui.Out().Printf("  after:   %s", canonicalWire)

	if dryRun {
		ui.Out().Print("dry-run: not writing")
		return
	}

	var rewritten bytes.Buffer
	rewritten.Grow(len(original))
	rewritten.Write(original[:valueStart])
	rewritten.WriteString(canonicalWire)
	rewritten.Write(original[valueEnd:])

	if cmd.backup {
		backupPath := fmt.Sprintf("%s.bak.%d", path, time.Now().Unix())
		if err := os.WriteFile(backupPath, original, 0o600); err != nil {
			errors.ContextCancelWithError(
				req,
				errors.Wrapf(err, "writing backup %s", backupPath),
			)
			return
		}
		ui.Out().Printf("wrote backup: %s", backupPath)
	}

	if err := os.WriteFile(path, rewritten.Bytes(), 0o600); err != nil {
		errors.ContextCancelWithError(
			req,
			errors.Wrapf(err, "writing %s", path),
		)
		return
	}

	ui.Out().Printf("rewrote %s", path)
}

func defaultConfigSeedPath() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "dodder", "config-seed"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrapf(err, "resolving user home dir")
	}

	return filepath.Join(home, ".local", "share", "dodder", "config-seed"), nil
}
