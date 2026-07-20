package config_log

import (
	"io"
	"testing"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func makeLog(t *ui.TestContext) (env_repo.Env, Log) {
	envRepo := env_repo.MakeTesting(t, nil)

	box := box_format.MakeBoxTransactedArchive(
		envRepo,
		options_print.Options{}.WithPrintTai(true),
	)
	closet := inventory_list_coders.MakeCloset(envRepo, box)

	return envRepo, Make(envRepo, closet)
}

// writeConfigBlob writes a small TOML blob to the default blob store and
// returns its digest, mirroring env_repo/testing.go's write-blob path.
func writeConfigBlob(
	t *ui.TestContext,
	envRepo env_repo.Env,
	content string,
) mad_domain_interfaces.MarklId {
	writeCloser, err := envRepo.GetDefaultBlobStore().MakeBlobWriter(nil)
	t.AssertNoError(err)

	reader, repool := pool.GetStringReader(content)
	defer repool()

	_, err = io.Copy(writeCloser, reader)
	t.AssertNoError(err)

	t.AssertNoError(writeCloser.Close())

	return writeCloser.GetMarklId()
}

func TestHeadEmpty(t1 *testing.T) {
	ui.RunTestContext(t1, testHeadEmpty)
}

func testHeadEmpty(t *ui.TestContext) {
	_, log := makeLog(t)

	_, _, err := log.Head()

	t.AssertTrue(
		errors.Is(err, ErrEmpty),
		"expected ErrEmpty on fresh repo",
	)
}

func TestAppendThenHead(t1 *testing.T) {
	ui.RunTestContext(t1, testAppendThenHead)
}

func testAppendThenHead(t *ui.TestContext) {
	envRepo, log := makeLog(t)

	blobDigest := writeConfigBlob(t, envRepo, "[blob-store]\n")
	configType := ids.MustType(ids.TypeTomlConfigV2)
	tai := ids.NowTai()

	t.AssertNoError(log.Append(blobDigest, configType, tai))

	head, repoolHead, err := log.Head()
	t.AssertNoError(err)
	defer repoolHead()

	t.AssertEqual("konfig", head.GetObjectId().String())

	// Regression guard: the entry must keep the config blob's own type
	// (!toml-config-v2), not the stream framing type (!inventory_list-v2).
	t.AssertEqual(ids.TypeTomlConfigV2, head.GetType().String())

	t.AssertNoError(markl.AssertEqual(blobDigest, head.GetBlobDigest()))

	t.AssertEqual(tai.String(), head.GetTai().String())

	t.AssertTrue(
		head.GetMetadata().GetMotherObjectSig().IsNull(),
		"expected root entry mother sig to be null",
	)

	t.AssertTrue(
		!head.GetMetadata().GetObjectSig().IsNull(),
		"expected object sig to be non-null (signed)",
	)
}

func TestAppendChains(t1 *testing.T) {
	ui.RunTestContext(t1, testAppendChains)
}

func testAppendChains(t *ui.TestContext) {
	envRepo, log := makeLog(t)

	firstDigest := writeConfigBlob(t, envRepo, "[blob-store]\nv = 1\n")
	secondDigest := writeConfigBlob(t, envRepo, "[blob-store]\nv = 2\n")
	configType := ids.MustType(ids.TypeTomlConfigV2)

	var firstObjectSig []byte

	{
		t.AssertNoError(log.Append(firstDigest, configType, ids.NowTai()))

		first, repoolFirst, err := log.Head()
		t.AssertNoError(err)

		firstObjectSig = append(
			firstObjectSig,
			first.GetMetadata().GetObjectSig().GetBytes()...,
		)

		repoolFirst()
	}

	t.AssertNoError(log.Append(secondDigest, configType, ids.NowTai()))

	head, repoolHead, err := log.Head()
	t.AssertNoError(err)
	defer repoolHead()

	t.AssertNoError(markl.AssertEqual(secondDigest, head.GetBlobDigest()))

	t.AssertEqual(
		firstObjectSig,
		head.GetMetadata().GetMotherObjectSig().GetBytes(),
	)

	{
		var count int

		for object, iterErr := range log.All() {
			t.AssertNoError(iterErr)
			t.AssertEqual("konfig", object.GetObjectId().String())
			count++
		}

		t.AssertEqual(2, count)
	}
}
