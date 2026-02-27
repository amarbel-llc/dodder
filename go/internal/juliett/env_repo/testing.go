//go:build test

package env_repo

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/debug"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
)

//go:noinline
func MakeTesting(
	t *ui.TestContext,
	contents map[string]string,
) (envRepo Env) {
	var bigBang BigBang
	bigBang.SetDefaults()

	return makeTestingWithBigBang(t, contents, bigBang)
}

//go:noinline
func makeTestingWithBigBang(
	t *ui.TestContext,
	contents map[string]string,
	bigBang BigBang,
) (envRepo Env) {
	t = t.Skip(2)

	dirTemp := t.TempDir()

	envDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
		t.Context,
		dirTemp,
		env_dir.XDGUtilityNameDodder,
		debug.Options{},
	)

	{
		var err error

		if envRepo, err = Make(
			env_local.Make(env_ui.MakeDefault(t.Context), envDir),
			Options{
				BasePath:                dirTemp,
				PermitNoDodderDirectory: true,
			},
		); err != nil {
			t.Errorf("failed to make repo: %s", err)
			return envRepo
		}
	}

	envRepo.Genesis(bigBang)

	if contents == nil {
		return envRepo
	}

	for expectedDigestString, content := range contents {
		var writeCloser domain_interfaces.BlobWriter

		writeCloser, err := envRepo.GetDefaultBlobStore().MakeBlobWriter(nil)
		if err != nil {
			errors.ContextCancelWithErrorAndFormat(
				t.Context,
				err,
				"failed to make blob writer",
			)
		}

		reader, repool := pool.GetStringReader(content)
		defer repool()
		_, err = io.Copy(writeCloser, reader)
		if err != nil {
			errors.ContextCancelWithErrorAndFormat(
				t.Context,
				err,
				"failed to write string to blob writer",
			)
		}

		err = writeCloser.Close()
		if err != nil {
			errors.ContextCancelWithErrorAndFormat(
				t.Context,
				err, "failed to write string to blob writer",
			)
		}

		actual := writeCloser.GetMarklId()
		t.Logf("actual blob digest: %q", actual)
		var expectedBlobDigest markl.Id
		t.AssertNoError(expectedBlobDigest.Set(expectedDigestString))
		t.AssertNoError(markl.AssertEqual(expectedBlobDigest, actual))
	}

	return envRepo
}
