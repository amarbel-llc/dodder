package quiter

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func AppendSeq[ELEMENT any, APPENDER interface{ Append(...ELEMENT) }](
	collection APPENDER,
	seq interfaces.Seq[ELEMENT],
) {
	for element := range seq {
		collection.Append(element)
	}
}

func AppendSeq2[INDEX any, ELEMENT any, APPENDER interface{ Append(...ELEMENT) }](
	collection APPENDER,
	seq interfaces.Seq2[INDEX, ELEMENT],
) {
	for _, element := range seq {
		collection.Append(element)
	}
}

func AddOrReplaceIfGreater[
	ELEMENT interface {
		interfaces.Stringer
	},
](
	set interfaces.SetMutable[ELEMENT],
	newElement ELEMENT,
	cmp cmp.Func[ELEMENT],
) (shouldAdd bool, err error) {
	existingElement, ok := set.Get(set.Key(newElement))

	shouldAdd = !ok || cmp(existingElement, newElement).IsLess()

	if shouldAdd {
		err = set.Add(newElement)
	}

	return shouldAdd, err
}

// Constructs an object of type `ELEMENT` by using its `Set` method and adds it to
// the given `adder`
func AddString[ELEMENT any, ELEMENT_PTR interfaces.SetterPtr[ELEMENT]](
	adder interfaces.Adder[ELEMENT],
	value string,
) (err error) {
	var element ELEMENT

	if err = ELEMENT_PTR(&element).Set(value); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = adder.Add(element); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
