package type_blobs

import (
	"fmt"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
)

//go:generate tommy generate
type FieldDefinition struct {
	Name    string   `toml:"name"`
	Kind    string   `toml:"kind"`
	Values  []string `toml:"values,omitempty"`
	Default string   `toml:"default,omitempty"`

	// Required rejects commits whose projected value for this field is
	// missing or empty, and rejects blobless commits of the declaring type
	// entirely (see oscar/store tryReadFields). omitempty keeps existing
	// blobs byte-identical: absent stays absent, false is never emitted.
	Required bool `toml:"required,omitempty"`

	// Terminal lists the enum values that make an object dormant: an object
	// whose projected value for this field is in Terminal is type-dormant
	// (FDR 0025). Only valid on enum fields, and every entry must also be in
	// Values. omitempty keeps existing blobs byte-identical.
	Terminal []string `toml:"terminal,omitempty"`

	// OmitEmpty makes an empty projected value read as unset (not projected),
	// as a no-default enum already does, instead of keeping the empty string.
	OmitEmpty bool `toml:"omit-empty,omitempty"`
}

// ValidateTerminal rejects a Terminal list on a non-enum field or one naming
// a value outside Values.
func (fd *FieldDefinition) ValidateTerminal() (err error) {
	if len(fd.Terminal) == 0 {
		return err
	}

	if fd.Kind != "enum" {
		return fmt.Errorf(
			"field %q: terminal is only valid on enum fields, not %q",
			fd.Name,
			fd.Kind,
		)
	}

	for _, value := range fd.Terminal {
		if !slices.Contains(fd.Values, value) {
			return fmt.Errorf(
				"field %q: terminal value %q is not in allowed values %v",
				fd.Name,
				value,
				fd.Values,
			)
		}
	}

	return err
}

func (fd *FieldDefinition) ToDefinition() fields.Definition {
	return fields.Definition{
		Name:    fd.Name,
		Kind:    fields.KindFromString(fd.Kind),
		Values:  fd.Values,
		Default: fd.Default,
	}
}
