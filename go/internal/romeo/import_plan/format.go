package import_plan

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func (plan *Plan) FormatSummary(w io.Writer) {
	p := message.NewPrinter(language.English)
	counts := plan.CountByClassification()

	p.Fprintf(w, "import plan: %d entries\n", len(plan.Entries))

	for _, c := range []Classification{
		ClassificationImport,
		ClassificationResolveTaiReassign,
		ClassificationSkipExists,
		ClassificationSkipDedup,
		ClassificationSkipBloblessType,
		ClassificationErrorMissingBlob,
	} {
		if n := counts[c]; n > 0 {
			p.Fprintf(w, "  %s: %d\n", c, n)
		}
	}

	committable := plan.CommittableCount()
	typeCount := plan.TypeCount()

	p.Fprintf(w, "committable: %d (%d types)\n", committable, typeCount)
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
