package commands_dodder

import (
	"bufio"
	"bytes"
	"io"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

func init() {
	// TEMPORARY one-time repair for a variant of dodder#358 (header-corrupted
	// config blob), reached this time via the LEGACY pre-FDR-0020 fallback
	// path (envRepo.FileConfig()'s stream-index-cached konfig sku) rather
	// than a config_log head, for repos that predate config_log entirely.
	// Same repair shape as this session's earlier repair-config-blob (since
	// deleted): read the raw blob via the unrestricted local env (before
	// SetBlobStoreOrder narrows it), strip a leading hyphence header if
	// present, write a corrected blob, then BOOTSTRAP a fresh config_log
	// with one entry pointing at it (preserving the original type/tai) --
	// since this store has no config_log to append onto, unlike the
	// earlier repair.
	utility.AddCmd("repair-legacy-config-blob", &RepairLegacyConfigBlob{})
}

type RepairLegacyConfigBlob struct {
	command_components_dodder.EnvRepo

	DryRun bool
}

var _ interfaces.CommandComponentWriter = (*RepairLegacyConfigBlob)(nil)

func (cmd RepairLegacyConfigBlob) GetDescription() command.Description {
	return command.Description{
		Short: "ONE-TIME: repair a header-corrupted legacy (pre-config_log) config blob and bootstrap config_log",
	}
}

func (cmd *RepairLegacyConfigBlob) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	f.BoolVar(&cmd.DryRun, "dry_run", false, "detect and verify without writing anything")
}

func (cmd RepairLegacyConfigBlob) Run(req command.Request) {
	env := cmd.MakeEnvRepo(req, false)

	if err := cmd.runRepair(env, cmd.DryRun); err != nil {
		env.Cancel(err)
	}
}

func (cmd RepairLegacyConfigBlob) runRepair(
	env env_repo.Env,
	dryRun bool,
) (err error) {
	// Step 1: read the legacy stream-index-cached konfig sku from
	// envRepo.FileConfig() ("config-mutable") to get its type + blob
	// digest + tai. This is the same fallback loadConfigSkuFromLog uses
	// when no config_log exists yet.
	file, err := files.Open(env.FileConfig())
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repoolReader := pool.GetBufferedReader(file)
	defer repoolReader()

	var coder stream_index.ListCoder
	var object sku.Transacted

	if _, err = coder.DecodeFrom(&object, bufferedReader); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var previousDigest markl.Id
	previousDigest.ResetWithMarklId(object.GetBlobDigest())

	configTypeStr := object.GetType().String()
	tai := object.GetTai()

	if !dryRun {
		if err = env.GetLockSmith().Lock(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer func() {
			if unlockErr := env.GetLockSmith().Unlock(); unlockErr != nil && err == nil {
				err = errors.Wrap(unlockErr)
			}
		}()
	}

	// Step 2: read the raw blob via the unrestricted local env (this
	// command never calls SetBlobStoreOrder, so ancestor/XDG discovery
	// stays unrestricted for the whole invocation).
	blobReader, err := env.GetLocalReadBlobStore().MakeBlobReader(&previousDigest)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobReader)

	rawBytes, err := io.ReadAll(blobReader)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	body, ok := stripLeadingHyphenceHeaderForLegacyRepair(rawBytes)
	if !ok {
		env.GetUI().Printf(
			"config blob %s has no leading hyphence header; nothing to repair (the malformed-value error must have a different cause)",
			previousDigest.String(),
		)
		return err
	}

	configType := ids.MustTypeStruct(configTypeStr)

	typedBlob := repo_configs.TypedBlob{Type: configType.ToMadder()}

	if _, err = repo_configs.Coder.Blob.DecodeFrom(
		&typedBlob,
		bufio.NewReader(bytes.NewReader(body)),
	); err != nil {
		err = errors.Wrapf(err, "stripped body still fails to decode; refusing to write")
		return err
	}

	env.GetUI().Printf(
		"detected leading hyphence header on legacy config blob %s (type %s, tai %s); stripped body decodes cleanly",
		previousDigest.String(),
		configTypeStr,
		tai.String(),
	)

	if dryRun {
		env.GetUI().Print("dry run: not writing a new blob or config_log entry")
		return err
	}

	// Step 3: write the corrected blob.
	var writer mad_domain_interfaces.BlobWriter

	if writer, err = env.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, writer)

	if _, err = writer.Write(body); err != nil {
		err = errors.Wrap(err)
		return err
	}

	newDigest := writer.GetMarklId()

	if markl.Equals(&previousDigest, newDigest) {
		env.GetUI().Print("new digest matches existing digest; nothing to append")
		return err
	}

	// Step 4: bootstrap a fresh config_log (this store has none) with one
	// entry pointing at the corrected blob, preserving the original type
	// and tai so history reads naturally.
	box := box_format.MakeBoxTransactedArchive(
		env,
		options_print.Options{}.WithPrintTai(true),
	)
	closet := inventory_list_coders.MakeCloset(env, box)
	cfgLog := config_log.Make(env, closet)

	if err = cfgLog.Append(newDigest, configType, tai); err != nil {
		err = errors.Wrap(err)
		return err
	}

	env.GetUI().Printf(
		"repaired legacy config blob and bootstrapped config_log: %s -> %s",
		previousDigest.String(),
		newDigest.String(),
	)

	return err
}

// stripLeadingHyphenceHeaderForLegacyRepair mirrors the earlier
// repair-config-blob's helper of the same shape (deleted after use
// earlier this session). Duplicated here rather than shared since both
// are one-time throwaway repair tools.
func stripLeadingHyphenceHeaderForLegacyRepair(raw []byte) (body []byte, ok bool) {
	const delim = "---\n"

	if !bytes.HasPrefix(raw, []byte(delim)) {
		return nil, false
	}

	parts := bytes.SplitN(raw, []byte(delim), 3)
	if len(parts) != 3 || len(parts[0]) != 0 {
		return nil, false
	}

	return bytes.TrimPrefix(parts[2], []byte("\n")), true
}
