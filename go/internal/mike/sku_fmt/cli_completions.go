package sku_fmt

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
)

type GenreObjectIdCollectionMap = map[genres.Genre]interfaces.Collection[string]

func OutputCliCompletions(
	envLocal env_local.Env,
	genreObjectIdCollections GenreObjectIdCollectionMap,
) {
	waitGroup := errors.MakeWaitGroupParallel()
	bufferedWriter, repool := pool.GetBufferedWriter(envLocal.GetUIFile())
	defer repool()

	defer errors.ContextMustFlush(envLocal, bufferedWriter)

	for genre, collection := range genreObjectIdCollections {
		waitGroup.Do(
			func() (err error) {
				for objectIdString := range collection.All() {
					bufferedWriter.WriteString(
						objectIdString,
					)
					bufferedWriter.WriteByte('\t')

					bufferedWriter.WriteString(genre.String())

					// tipe := objectIdString.GetType().String()

					// if tipe != "" {
					// 	bufferedWriter.WriteString(": ")
					// 	bufferedWriter.WriteString(
					// 		objectIdString.GetType().String(),
					// 	)
					// }

					// description :=
					// objectIdString.GetMetadata().Description.String()

					// if description != "" {
					// 	bufferedWriter.WriteString(" ")
					// 	bufferedWriter.WriteString(
					// 		objectIdString.GetMetadata().Description.String(),
					// 	)
					// }

					bufferedWriter.WriteString("\n")

					return err
				}

				return err
			},
		)
	}

	envLocal.Must(errors.MakeFuncContextFromFuncErr(waitGroup.GetError))
}
