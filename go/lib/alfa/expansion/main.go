package expansion

import (
	"slices"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/0/quiter_seq"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var (
	ExpanderRight = MakeExpanderRight(`-`)
	ExpanderAll   = MakeExpanderAll(`-`)
)

type Expander interface {
	Expand(string) interfaces.Seq[string]
}

func ExpandOneIntoIds[
	ID interfaces.Value,
	ID_PTR interfaces.ValuePtr[ID],
](
	identifierString string,
	expander Expander,
) interfaces.SeqError[ID] {
	return func(yield func(ID, error) bool) {
		if identifierString == "" {
			return
		}

		var expandedId ID

		for expanded := range expander.Expand(identifierString) {
			if expanded == "" {
				continue
			}

			if err := ID_PTR(&expandedId).Set(expanded); err != nil {
				if !yield(expandedId, err) {
					return
				}
			}

			if !yield(expandedId, nil) {
				return
			}
		}
	}
}

func ExpandIntoSlice[
	ID interfaces.Value,
	ID_PTR interfaces.ValuePtr[ID],
](
	token string,
	expander Expander,
) collections_slice.Slice[ID] {
	return slices.Collect(
		quiter_seq.SeqErrorToSeqAndPanic(
			ExpandOneIntoIds[ID, ID_PTR](
				token,
				expander,
			),
		),
	)
}

func ExpandMany[
	ID interfaces.Value,
	ID_PTR interfaces.ValuePtr[ID],
](
	seq interfaces.Seq[ID],
	expander Expander,
) interfaces.Seq[ID] {
	return func(yield func(ID) bool) {
		for id := range seq {
			seqExpansion := quiter_seq.SeqErrorToSeqAndPanic(
				ExpandOneIntoIds[ID, ID_PTR](id.String(), expander),
			)

			for expanded := range seqExpansion {
				if !yield(expanded) {
					return
				}
			}
		}
	}
}
