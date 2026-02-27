package interfaces

import (
	"iter"
)

type (
	Seq[ELEMENT any]                   = iter.Seq[ELEMENT]
	Seq2[ELEMENT any, ELEMENT_TWO any] = iter.Seq2[ELEMENT, ELEMENT_TWO]
	SeqError[ELEMENT any]              = iter.Seq2[ELEMENT, error]

	Pull[ELEMENT any]                   = func() (ELEMENT, bool)
	Pull2[ELEMENT any, ELEMENT_TWO any] = func() (ELEMENT, ELEMENT_TWO, bool)
)
