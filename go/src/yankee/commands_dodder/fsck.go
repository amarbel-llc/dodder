package commands_dodder

import (
	"io"
	"sync/atomic"
	"time"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/hotel/object_fmt_digest"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/object_finalizer"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd(
		"fsck",
		&Fsck{
			VerifyOptions: object_finalizer.DefaultVerifyOptions(),
		},
	)
}

// TODO add options to verify type formats, tags
// TODO add option to count duplicate objects according to a list of object
// digest formats
type Fsck struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.InventoryLists
	command_components_dodder.Query

	InventoryListPath string

	VerifyOptions object_finalizer.VerifyOptions
	Duplicates    object_fmt_digest.CLIFlag
	SkipProbes    bool
	SkipBlobs     bool
}

var _ interfaces.CommandComponentWriter = (*Fsck)(nil)

func (cmd *Fsck) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&cmd.InventoryListPath,
		"inventory_list-path",
		"",
		"instead of using the store's object, verify the objects at the inventory list at the given path",
	)

	flagSet.BoolVar(
		&cmd.VerifyOptions.ObjectSigPresent,
		"object-sig-required",
		true,
		"require the object signature when validating",
	)

	flagSet.BoolVar(
		&cmd.SkipProbes,
		"skip-probes",
		false,
		"skip verification of probe index entries",
	)

	flagSet.BoolVar(
		&cmd.SkipBlobs,
		"skip-blobs",
		false,
		"skip verification of blob contents",
	)

	cmd.Duplicates.SetFlagDefinitions(flagSet)
}

func (cmd Fsck) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	var seq interfaces.SeqError[*sku.Transacted]

	if cmd.InventoryListPath == "" {
		query := cmd.MakeQuery(
			req,
			queries.BuilderOptions(
				queries.BuilderOptionDefaultGenres(genres.All()...),
				queries.BuilderOptionDefaultSigil(
					ids.SigilLatest,
					ids.SigilHistory,
					ids.SigilHidden,
				),
			),
			repo,
			req.PopArgs(),
		)

		seq = repo.GetStore().All(query)

		ui.Out().Printf("verification for %q objects in progress...", query)
	} else {
		seq = cmd.MakeSeqFromPath(
			repo,
			repo.GetInventoryListCoderCloset(),
			cmd.InventoryListPath,
			nil,
		)
	}

	cmd.runVerification(repo, seq)
}

func (cmd Fsck) runVerification(
	repo *local_working_copy.Repo,
	seq interfaces.SeqError[*sku.Transacted],
) {
	var count atomic.Uint32

	type objectError struct {
		object *sku.Transacted
		err    error
	}

	var objectErrors collections_slice.Slice[objectError]

	finalizer := object_finalizer.Builder().
		WithVerifyOptions(cmd.VerifyOptions).
		Build()

	if err := errors.RunChildContextWithPrintTicker(
		repo,
		func(ctx errors.Context) {
			for object, errIter := range seq {
				if errIter != nil {
					err := objectError{err: errIter}

					if object != nil {
						cloned, _ := object.CloneTransacted()
						err.object = cloned
					}

					objectErrors.Append(err)

					continue
				}

				if err := markl.AssertIdIsNotNull(
					object.GetObjectDigest(),
				); err != nil {
					cloned, _ := object.CloneTransacted()
					objectErrors.Append(
						objectError{
							err:    err,
							object: cloned,
						},
					)
				}

				if err := finalizer.Verify(object); err != nil {
					cloned, _ := object.CloneTransacted()
					objectErrors.Append(
						objectError{
							err:    err,
							object: cloned,
						},
					)
				}

				if !cmd.SkipProbes {
					if err := repo.GetStore().GetStreamIndex().VerifyObjectProbes(
						object,
					); err != nil {
						cloned, _ := object.CloneTransacted()
						objectErrors.Append(
							objectError{
								err:    err,
								object: cloned,
							},
						)
					}
				}

				if !cmd.SkipBlobs {
					blobDigest := object.GetBlobDigest()
					if !blobDigest.IsNull() {
						if err := blob_stores.VerifyBlob(
							repo,
							repo.GetEnvRepo().GetDefaultBlobStore(),
							blobDigest,
							io.Discard,
						); err != nil {
							cloned, _ := object.CloneTransacted()
							objectErrors.Append(
								objectError{
									err:    errors.Wrapf(err, "blob verification failed"),
									object: cloned,
								},
							)
						}
					}
				}

				count.Add(1)
			}
		},
		func(time time.Time) {
			ui.Out().Printf(
				"(in progress) %d verified, %d errors",
				count.Load(),
				len(objectErrors),
			)
		},
		3*time.Second,
	); err != nil {
		repo.Cancel(err)
		return
	}

	ui.Out().Printf("verification complete")
	ui.Out().Printf("objects verified: %d", count.Load())
	ui.Out().Printf("objects with errors: %d", len(objectErrors))

	for _, objectError := range objectErrors {
		ui.Out().Printf("%s:", sku.StringMetadataTaiMerkle(objectError.object))
		ui.CLIErrorTreeEncoder.EncodeTo(objectError.err, ui.Out())
	}
}
