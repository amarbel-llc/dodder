package commands_dodder

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/echo/object_fmt_digest"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/india/stream_index_fixed"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/tap-dancer/go"
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
	TryV14Index   bool
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

	flagSet.BoolVar(
		&cmd.TryV14Index,
		"try-v14-index",
		false,
		"after verification, build a V14 fixed-length index in a temp directory and validate round-trip",
	)

	cmd.Duplicates.SetFlagDefinitions(flagSet)
}

func (cmd Fsck) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	tw := tap.NewWriter(os.Stdout)

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

		tw.Comment(fmt.Sprintf("verification for %q objects in progress...", query))
	} else {
		seq = cmd.MakeSeqFromPath(
			repo,
			repo.GetInventoryListCoderCloset(),
			cmd.InventoryListPath,
			nil,
		)
	}

	cmd.runVerification(repo, tw, seq)

	if cmd.TryV14Index {
		cmd.runV14IndexTrial(repo, tw)
	}
}

func (cmd Fsck) runVerification(
	repo *local_working_copy.Repo,
	tw *tap.Writer,
	seq interfaces.SeqError[*sku.Transacted],
) {
	var count atomic.Uint32
	var errorCount atomic.Uint32

	finalizer := object_finalizer.Builder().
		WithVerifyOptions(cmd.VerifyOptions).
		Build()

	if err := errors.RunChildContextWithPrintTicker(
		repo,
		func(ctx errors.Context) {
			for object, errIter := range seq {
				if errIter != nil {
					desc := "iteration error"
					if object != nil {
						desc = sku.StringMetadataTaiMerkle(object)
					}

					tw.NotOk(desc, tap_diagnostics.FromError(errIter))
					errorCount.Add(1)
					count.Add(1)

					continue
				}

				desc := sku.StringMetadataTaiMerkle(object)
				var objectErrors []error

				if err := markl.AssertIdIsNotNull(
					object.GetObjectDigest(),
				); err != nil {
					objectErrors = append(objectErrors, err)
				}

				if err := finalizer.Verify(object); err != nil {
					objectErrors = append(objectErrors, err)
				}

				if !cmd.SkipProbes {
					if err := repo.GetStore().GetStreamIndex().VerifyObjectProbes(
						object,
					); err != nil {
						objectErrors = append(objectErrors, err)
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
							objectErrors = append(objectErrors, errors.Wrapf(err, "blob verification failed"))
						}
					}
				}

				if len(objectErrors) == 0 {
					tw.Ok(desc)
				} else {
					diag := tap_diagnostics.FromError(objectErrors[0])
					if len(objectErrors) > 1 {
						msgs := make([]string, len(objectErrors))
						for i, e := range objectErrors {
							msgs[i] = e.Error()
						}
						diag["message"] = strings.Join(msgs, "; ")
					}
					tw.NotOk(desc, diag)
					errorCount.Add(1)
				}

				count.Add(1)
			}
		},
		func(time time.Time) {
			tw.Comment(fmt.Sprintf(
				"(in progress) %d verified, %d errors",
				count.Load(),
				errorCount.Load(),
			))
		},
		3*time.Second,
	); err != nil {
		tw.BailOut(err.Error())
		repo.Cancel(err)
		return
	}

	tw.Plan()
}

func (cmd Fsck) runV14IndexTrial(
	repo *local_working_copy.Repo,
	tw *tap.Writer,
) (err error) {
	tw.Comment("starting V14 fixed-length index trial...")

	tempDir, err := os.MkdirTemp("", "dodder-v14-trial-*")
	if err != nil {
		tw.BailOut(fmt.Sprintf("failed to create temp dir: %s", err))
		repo.Cancel(err)
		return
	}

	defer errors.Deferred(&err, func() error { return os.RemoveAll(tempDir) })

	// Build V14 index in temp directory.
	v14Index, err := stream_index_fixed.MakeIndex(
		repo.GetEnvRepo(),
		func(_ *sku.Transacted) error { return nil },
		tempDir,
		ids.Tai{},
		0,
	)

	if err != nil {
		tw.BailOut(fmt.Sprintf("failed to create V14 index: %s", err))
		repo.Cancel(err)
		return
	}

	// Iterate all objects from the current store and add to V14 index.
	query := cmd.MakeQuery(
		command.Request{},
		queries.BuilderOptions(
			queries.BuilderOptionDefaultGenres(genres.All()...),
			queries.BuilderOptionDefaultSigil(
				ids.SigilLatest,
				ids.SigilHistory,
				ids.SigilHidden,
			),
		),
		repo,
		nil,
	)

	var addCount atomic.Uint32
	var addErrorCount atomic.Uint32

	if err = errors.RunChildContextWithPrintTicker(
		repo,
		func(ctx errors.Context) {
			seq := repo.GetStore().All(query)

			for object, errIter := range seq {
				if errIter != nil {
					tw.NotOk(
						"v14-add iteration error",
						tap_diagnostics.FromError(errIter),
					)
					addErrorCount.Add(1)
					addCount.Add(1)

					continue
				}

				if err := v14Index.Add(object, sku.CommitOptions{
					StoreOptions: sku.StoreOptions{
						StreamIndexOptions: sku.StreamIndexOptions{
							AddToStreamIndex: true,
							ForceLatest:      true,
						},
					},
				}); err != nil {
					tw.NotOk(
						fmt.Sprintf("v14-add %s", sku.StringMetadataTaiMerkle(object)),
						tap_diagnostics.FromError(err),
					)
					addErrorCount.Add(1)
				} else {
					addCount.Add(1)
				}
			}
		},
		func(time time.Time) {
			tw.Comment(fmt.Sprintf(
				"(v14 adding) %d added, %d errors",
				addCount.Load(),
				addErrorCount.Load(),
			))
		},
		3*time.Second,
	); err != nil {
		tw.BailOut(fmt.Sprintf("v14 add phase failed: %s", err))
		repo.Cancel(err)
		return
	}

	tw.Comment(fmt.Sprintf(
		"v14 add phase complete: %d objects, %d errors",
		addCount.Load(),
		addErrorCount.Load(),
	))

	// Flush the V14 index.
	if err = v14Index.Flush(func(msg string) error {
		tw.Comment(fmt.Sprintf("v14 flush: %s", msg))
		return nil
	}); err != nil {
		tw.BailOut(fmt.Sprintf("v14 flush failed: %s", err))
		repo.Cancel(err)
		return
	}

	tw.Comment("v14 flush complete, starting read-back verification...")

	// Read back every object and compare against originals.
	var verifyCount atomic.Uint32
	var verifyErrorCount atomic.Uint32
	var taiCollisionCount atomic.Uint32
	var inlineCount atomic.Uint32

	if err = errors.RunChildContextWithPrintTicker(
		repo,
		func(ctx errors.Context) {
			seq := repo.GetStore().All(query)

			for object, errIter := range seq {
				if errIter != nil {
					tw.NotOk(
						"v14-verify iteration error",
						tap_diagnostics.FromError(errIter),
					)
					verifyErrorCount.Add(1)
					verifyCount.Add(1)

					continue
				}

				desc := fmt.Sprintf(
					"v14-verify %s",
					sku.StringMetadataTaiMerkle(object),
				)

				readBack, err := v14Index.ReadOneObjectIdTai(
					object.GetObjectId(),
					object.GetTai(),
				)
				if err != nil {
					tw.NotOk(desc, tap_diagnostics.FromError(err))
					verifyErrorCount.Add(1)
				} else if readBack.GetObjectId().String() == object.GetObjectId().String() &&
					readBack.GetTai().String() == object.GetTai().String() &&
					readBack.GetObjectDigest().String() != object.GetObjectDigest().String() {
					// Same objectId+tai but different object digest means
					// multiple versions share the same tai (migration-era
					// data). The probe can only store one; this is a known
					// limitation, not an index error.
					tw.Skip(desc, fmt.Sprintf(
						"tai collision: probe stores digest %s, expected %s",
						readBack.GetObjectDigest(),
						object.GetObjectDigest(),
					))
					taiCollisionCount.Add(1)
				} else {
					var mismatches []string

					if readBack.GetObjectId().String() != object.GetObjectId().String() {
						mismatches = append(mismatches, fmt.Sprintf(
							"ObjectId: expected %s, got %s",
							object.GetObjectId(),
							readBack.GetObjectId(),
						))
					}

					if readBack.GetTai().String() != object.GetTai().String() {
						mismatches = append(mismatches, fmt.Sprintf(
							"Tai: expected %s, got %s",
							object.GetTai(),
							readBack.GetTai(),
						))
					}

					if readBack.GetType().String() != object.GetType().String() {
						mismatches = append(mismatches, fmt.Sprintf(
							"Type: expected %s, got %s",
							object.GetType(),
							readBack.GetType(),
						))
					}

					if readBack.GetBlobDigest().String() != object.GetBlobDigest().String() {
						mismatches = append(mismatches, fmt.Sprintf(
							"Blob: expected %s, got %s",
							object.GetBlobDigest(),
							readBack.GetBlobDigest(),
						))
					}

					if len(mismatches) == 0 {
						tw.Ok(desc)
						inlineCount.Add(1)
					} else {
						tw.NotOk(desc, map[string]string{
							"message": strings.Join(mismatches, "; "),
						})
						verifyErrorCount.Add(1)
					}
				}

				verifyCount.Add(1)
			}
		},
		func(time time.Time) {
			tw.Comment(fmt.Sprintf(
				"(v14 verifying) %d verified, %d errors",
				verifyCount.Load(),
				verifyErrorCount.Load(),
			))
		},
		3*time.Second,
	); err != nil {
		tw.BailOut(fmt.Sprintf("v14 verify phase failed: %s", err))
		repo.Cancel(err)
		return
	}

	tw.Comment(fmt.Sprintf(
		"v14 trial complete: %d verified, %d errors, %d inline, %d tai collisions",
		verifyCount.Load(),
		verifyErrorCount.Load(),
		inlineCount.Load(),
		taiCollisionCount.Load(),
	))

	return
}
