package type_blobs

import "code.linenisgreat.com/dodder/go/internal/0/fields"

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
}

func (fd *FieldDefinition) ToDefinition() fields.Definition {
	return fields.Definition{
		Name:    fd.Name,
		Kind:    fields.KindFromString(fd.Kind),
		Values:  fd.Values,
		Default: fd.Default,
	}
}
