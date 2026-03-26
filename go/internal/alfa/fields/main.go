package fields

type Type string

const (
	TypeNormal   Type = ""
	TypeId       Type = "\u001b[34m"
	TypeHash     Type = "\u001b[3m"
	TypeError    Type = "\u001b[31m"
	TypeType     Type = "\u001b[33m"
	TypeUserData Type = "\u001b[36m"
	TypeHeading  Type = "\u001b[31m"
)

type Field struct {
	Type
	Key, Value string
}
