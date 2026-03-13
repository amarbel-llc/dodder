package import_plan

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

func (plan *Plan) FormatSummary(w io.Writer) {
	counts := plan.CountByClassification()

	fmt.Fprintf(w, "import plan: %d entries\n", len(plan.Entries))

	for _, c := range []Classification{
		ClassificationImport,
		ClassificationResolveTaiReassign,
		ClassificationSkipExists,
		ClassificationSkipDedup,
		ClassificationSkipBloblessType,
		ClassificationErrorMissingBlob,
	} {
		if n := counts[c]; n > 0 {
			fmt.Fprintf(w, "  %s: %d\n", c, n)
		}
	}

	committable := plan.CommittableCount()
	typeCount := plan.TypeCount()

	fmt.Fprintf(w, "committable: %d (%d types)\n", committable, typeCount)
}

func (plan *Plan) FormatObjects(w io.Writer) {
	for i := range plan.Entries {
		entry := &plan.Entries[i]

		genre := genres.Make(entry.object.GetGenre())
		objectId := sku.String(&entry.object)
		tai := entry.object.GetTai()

		if entry.Classification == ClassificationResolveTaiReassign {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s -> %s\n",
				entry.Classification,
				genre,
				objectId,
				entry.OriginalTai,
				tai,
			)
		} else {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\n",
				entry.Classification,
				genre,
				objectId,
				tai,
			)
		}
	}
}
