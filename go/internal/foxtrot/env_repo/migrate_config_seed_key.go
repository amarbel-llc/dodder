package env_repo

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

var reConfigSeedPrivateKey = regexp.MustCompile(
	`(?m)^(private-key\s*=\s*['"])([^'"]+)(['"])`,
)

// ConfigSeedMigrationResult records the outcome of a single config-seed
// private-key re-encode pass.
type ConfigSeedMigrationResult struct {
	Path          string
	Purpose       string
	FormatId      string
	DataBytes     int
	LegacyWire    string
	CanonicalWire string
	BackupPath    string
	Migrated      bool
	AlreadyCanonical bool
}

// MigrateLegacyCombinedHRPConfigSeed re-encodes the `private-key`
// markl-id in the config-seed file at path from the legacy
// combined-HRP wire form (pre-madder v0.3.16) to the canonical
// split-HRP form. The underlying key bytes are preserved; only the
// trailing blech32 checksum changes.
//
// AlreadyCanonical=true is returned without error when the file is
// already in canonical form. backup=true writes a .bak.<unix-seconds>
// sibling before overwriting; dryRun=true returns the proposed change
// without writing anything (BackupPath stays empty).
func MigrateLegacyCombinedHRPConfigSeed(
	path string,
	backup bool,
	dryRun bool,
) (ConfigSeedMigrationResult, error) {
	result := ConfigSeedMigrationResult{Path: path}

	original, err := os.ReadFile(path)
	if err != nil {
		return result, errors.Wrapf(err, "reading %s", path)
	}

	match := reConfigSeedPrivateKey.FindSubmatchIndex(original)
	if match == nil {
		return result, errors.Errorf(
			"no private-key field found in %s",
			path,
		)
	}

	valueStart, valueEnd := match[4], match[5]
	legacyWire := string(original[valueStart:valueEnd])
	result.LegacyWire = legacyWire

	var probe markl.Id
	decodeErr := probe.UnmarshalText([]byte(legacyWire))
	if decodeErr == nil {
		result.AlreadyCanonical = true
		return result, nil
	}

	var legacy markl.ErrLegacyCombinedHRPWireForm
	if !errors.As(decodeErr, &legacy) {
		return result, errors.Wrapf(
			decodeErr,
			"decoding private-key in %s",
			path,
		)
	}

	result.Purpose = legacy.Purpose
	result.FormatId = legacy.FormatId
	result.DataBytes = len(legacy.Data)

	var canonical markl.Id
	if err := canonical.SetPurposeId(legacy.Purpose); err != nil {
		return result, errors.Wrapf(
			err,
			"setting purpose %q",
			legacy.Purpose,
		)
	}

	if err := canonical.SetMarklId(legacy.FormatId, legacy.Data); err != nil {
		return result, errors.Wrapf(
			err,
			"setting format=%q data=%d bytes",
			legacy.FormatId,
			len(legacy.Data),
		)
	}

	canonicalBytes, err := canonical.MarshalText()
	if err != nil {
		return result, errors.Wrapf(err, "re-marshaling private-key")
	}

	canonicalWire := string(canonicalBytes)
	result.CanonicalWire = canonicalWire

	if canonicalWire == legacyWire {
		return result, errors.Errorf(
			"re-encode produced identical wire form; refusing to write (purpose=%q, format=%q)",
			legacy.Purpose,
			legacy.FormatId,
		)
	}

	if dryRun {
		return result, nil
	}

	var rewritten bytes.Buffer
	rewritten.Grow(len(original))
	rewritten.Write(original[:valueStart])
	rewritten.WriteString(canonicalWire)
	rewritten.Write(original[valueEnd:])

	if backup {
		backupPath := fmt.Sprintf("%s.bak.%d", path, time.Now().Unix())
		if err := os.WriteFile(backupPath, original, 0o600); err != nil {
			return result, errors.Wrapf(
				err,
				"writing backup %s",
				backupPath,
			)
		}
		result.BackupPath = backupPath
	}

	if err := os.WriteFile(path, rewritten.Bytes(), 0o600); err != nil {
		return result, errors.Wrapf(err, "writing %s", path)
	}

	result.Migrated = true
	return result, nil
}
