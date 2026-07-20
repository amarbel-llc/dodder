package fields

import "code.linenisgreat.com/piggy/go/pkgs/markl"

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

type Kind byte

const (
	KindString     Kind = iota // WIT: string
	KindEnum                   // WIT: enum
	KindBool                   // WIT: bool
	KindU32                    // WIT: u32
	KindS32                    // WIT: s32
	KindListString             // WIT: list<string>
)

func KindFromString(s string) Kind {
	switch s {
	case "string":
		return KindString
	case "enum":
		return KindEnum
	case "bool":
		return KindBool
	case "u32":
		return KindU32
	case "s32":
		return KindS32
	case "list<string>":
		return KindListString
	default:
		return KindString
	}
}

type Definition struct {
	Name    string
	Kind    Kind
	Values  []string // populated for KindEnum
	Default string
}

type Field struct {
	Type
	Key, Value     string
	TypeBlobDigest markl.Id // lookup hint for lazy definition resolution
}
