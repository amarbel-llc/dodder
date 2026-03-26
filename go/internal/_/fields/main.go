package fields

type Type byte

const (
	TypeNormal   Type = iota
	TypeId            // object and zettel identifiers
	TypeHash          // content-addressable digests and signatures
	TypeError         // error messages
	TypeType          // object type identifiers
	TypeUserData      // user-provided content (descriptions, tags values)
	TypeHeading       // section headings
)

type Field struct {
	Type
	Key, Value string
}
