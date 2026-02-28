package ids

import "code.linenisgreat.com/dodder/go/lib/delta/catgut"

type Field struct {
	key, value catgut.String
}

func (f *Field) SetCatgutString(v *catgut.String) (err error) {
	return err
}
