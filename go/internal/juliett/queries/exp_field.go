package queries

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
)

type expField struct {
	Key   string
	Value string
}

func (e *expField) ContainsSku(
	objectGetter sku.TransactedGetter,
) bool {
	object := objectGetter.GetSku()
	metadata := object.GetMetadata()

	for field := range metadata.GetIndex().GetFields() {
		if e.matchesField(field) {
			return true
		}
	}

	return false
}

func (e *expField) matchesField(field fields.Field) bool {
	return field.Key == e.Key && field.Value == e.Value
}

func (e *expField) String() string {
	return fmt.Sprintf("%s=%s", e.Key, e.Value)
}
