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

	plan.formatErrorTree(w)
}

func (plan *Plan) formatErrorTree(w io.Writer) {
	type errorRoot struct {
		entry      *Entry
		dependents []*Entry
	}

	roots := make(map[string]*errorRoot)
	var rootOrder []string

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsError() &&
			entry.Classification != ClassificationSkipBloblessType {
			continue
		}

		if entry.ErrorCause != "" {
			continue
		}

		genre := genres.Make(entry.object.GetGenre())
		if genre != genres.Type {
			key := sku.String(&entry.object)
			roots[key] = &errorRoot{entry: entry}
			rootOrder = append(rootOrder, key)
			continue
		}

		key := entry.object.GetObjectId().String()
		roots[key] = &errorRoot{entry: entry}
		rootOrder = append(rootOrder, key)
	}

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if entry.ErrorCause == "" {
			continue
		}

		if root, ok := roots[entry.ErrorCause]; ok {
			root.dependents = append(root.dependents, entry)
		}
	}

	if len(rootOrder) == 0 {
		return
	}

	fmt.Fprintln(w, "errors:")

	for ri, key := range rootOrder {
		root := roots[key]
		isLastRoot := ri == len(rootOrder)-1

		var prefix, childPrefix string
		if isLastRoot {
			prefix = "  └── "
			childPrefix = "      "
		} else {
			prefix = "  ├── "
			childPrefix = "  │   "
		}

		genre := genres.Make(root.entry.object.GetGenre())
		objectId := root.entry.object.GetObjectId().String()

		fmt.Fprintf(w, "%s%s %s %s\n",
			prefix,
			root.entry.Classification,
			genre,
			objectId,
		)

		for di, dep := range root.dependents {
			isLastDep := di == len(root.dependents)-1

			var depPrefix string
			if isLastDep {
				depPrefix = childPrefix + "└── "
			} else {
				depPrefix = childPrefix + "├── "
			}

			objectId := dep.object.GetObjectId().String()
			description := dep.object.GetMetadata().GetDescription().String()

			if description != "" {
				fmt.Fprintf(w, "%s%s %q\n", depPrefix, objectId, description)
			} else {
				fmt.Fprintf(w, "%s%s\n", depPrefix, objectId)
			}
		}
	}
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
