package import_plan

import (
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var classificationOrder = []Classification{
	ClassificationImport,
	ClassificationResolveTaiReassign,
	ClassificationSkipExists,
	ClassificationSkipDedup,
	ClassificationSkipBloblessType,
	ClassificationErrorMissingBlob,
}

func (plan *Plan) FormatSummary(
	w io.Writer,
	boxFormatter *box_format.BoxTransacted,
) {
	p := message.NewPrinter(language.English)
	counts := plan.CountByClassification()

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Faint(true)).
		Headers("classification", "count").
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

			if col == 1 {
				s = s.Align(lipgloss.Right)
			}

			return s
		})

	for _, c := range classificationOrder {
		if n := counts[c]; n > 0 {
			t.Row(string(c), p.Sprintf("%d", n))
		}
	}

	committable := plan.CommittableCount()
	typeCount := plan.TypeCount()

	t.Row("committable", p.Sprintf("%d (%d types)", committable, typeCount))

	fmt.Fprintln(w, t.Render())

	plan.formatErrorTree(w, boxFormatter)
}

func (plan *Plan) formatErrorTree(
	w io.Writer,
	boxFormatter *box_format.BoxTransacted,
) {
	type errorRoot struct {
		entry      *Entry
		dependents []*Entry
	}

	type classificationGroup struct {
		classification Classification
		roots          []*errorRoot
	}

	roots := make(map[string]*errorRoot)
	groups := make(map[Classification]*classificationGroup)
	var groupOrder []Classification

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsError() &&
			entry.Classification != ClassificationSkipBloblessType {
			continue
		}

		if entry.ErrorCause != "" {
			continue
		}

		var key string

		genre := genres.Make(entry.object.GetGenre())
		if genre != genres.Type {
			key = sku.String(&entry.object)
		} else {
			key = entry.object.GetObjectId().String()
		}

		root := &errorRoot{entry: entry}
		roots[key] = root

		group, ok := groups[entry.Classification]
		if !ok {
			group = &classificationGroup{classification: entry.Classification}
			groups[entry.Classification] = group
			groupOrder = append(groupOrder, entry.Classification)
		}

		group.roots = append(group.roots, root)
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

	if len(groupOrder) == 0 {
		return
	}

	fmt.Fprintln(w, "errors:")

	for gi, classification := range groupOrder {
		group := groups[classification]
		isLastGroup := gi == len(groupOrder)-1

		var groupPrefix, groupChildPrefix string
		if isLastGroup {
			groupPrefix = "  └── "
			groupChildPrefix = "      "
		} else {
			groupPrefix = "  ├── "
			groupChildPrefix = "  │   "
		}

		fmt.Fprintf(w, "%s%s\n", groupPrefix, classification)

		for ri, root := range group.roots {
			isLastRoot := ri == len(group.roots)-1

			var rootPrefix, rootChildPrefix string
			if isLastRoot {
				rootPrefix = groupChildPrefix + "└── "
				rootChildPrefix = groupChildPrefix + "    "
			} else {
				rootPrefix = groupChildPrefix + "├── "
				rootChildPrefix = groupChildPrefix + "│   "
			}

			genre := genres.Make(root.entry.object.GetGenre())
			objectId := root.entry.object.GetObjectId().String()

			fmt.Fprintf(w, "%s%s %s\n", rootPrefix, genre, objectId)

			for di, dep := range root.dependents {
				isLastDep := di == len(root.dependents)-1

				var depPrefix string
				if isLastDep {
					depPrefix = rootChildPrefix + "└── "
				} else {
					depPrefix = rootChildPrefix + "├── "
				}

				sb := &strings.Builder{}
				boxFormatter.EncodeStringTo(&dep.object, sb)

				fmt.Fprintf(w, "%s%s\n", depPrefix, sb.String())
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
