package store_config

import (
	"bufio"
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// TestBootstrapReadsConfigLogHead is a diagnostic: it appends a config
// state to the config log, then bootstraps a fresh store_config and
// asserts the loaded config Sku's blob digest equals the appended blob
// digest. If bootstrap silently fell back (or swallowed an error), the
// loaded Sku would be empty and this fails.
func TestBootstrapReadsConfigLogHead(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		envRepo := env_repo.MakeTesting(t, nil)

		box := box_format.MakeBoxTransactedArchive(
			envRepo,
			options_print.Options{}.WithPrintTai(true),
		)
		closet := inventory_list_coders.MakeCloset(envRepo, box)
		cfgLog := config_log.Make(envRepo, closet)

		// A distinctive default type so we can prove the config VALUES (not
		// just the Sku metadata) were decoded from the log blob. Initialize
		// swallows IsNotExist, so a blob that failed to load would silently
		// leave defaults in place — asserting this value catches that.
		distinctiveType := ids.MustTypeStruct("!custom-default-type")
		typedBlob := repo_configs.DefaultOverlay(nil, distinctiveType)

		var blobDigest markl.Id

		{
			blobWriter, err := envRepo.GetDefaultBlobStore().MakeBlobWriter(nil)
			t.AssertNoError(err)

			bufferedWriter, repool := pool.GetBufferedWriter(blobWriter)

			_, err = repo_configs.Coder.Blob.EncodeTo(&typedBlob, bufferedWriter)
			t.AssertNoError(err)

			t.AssertNoError(bufferedWriter.Flush())
			repool()

			t.AssertNoError(blobWriter.Close())
			blobDigest.ResetWithMarklId(blobWriter.GetMarklId())
		}

		t.AssertNoError(
			cfgLog.Append(
				&blobDigest,
				ids.MustType(ids.TypeTomlConfigV2),
				ids.NowTai(),
			),
		)

		store := Make()
		t.AssertNoError(store.Initialize(envRepo, repo_config_cli.Config{}))

		loadedDigest := store.GetConfig().GetSku().GetBlobDigest()
		t.AssertNoError(markl.AssertEqual(&blobDigest, loadedDigest))

		// The decoded config values must reflect the log blob, not defaults.
		t.AssertEqual(
			distinctiveType.String(),
			store.GetConfig().GetDefaults().GetDefaultType().String(),
		)
	})
}

func TestListCoderRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	ta, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	t.AssertNoError(ta.GetObjectIdMutable().Set("test-tag"))

	var buf bytes.Buffer
	var coder stream_index.ListCoder

	writer := bufio.NewWriter(&buf)

	_, err := coder.EncodeTo(ta, writer)
	t.AssertNoError(err)

	t.AssertNoError(writer.Flush())

	reader := bufio.NewReader(&buf)

	var actual sku.Transacted

	_, err = coder.DecodeFrom(&actual, reader)
	t.AssertNoError(err)

	t.AssertEqual(ta.GetObjectId().String(), actual.GetObjectId().String())
}
