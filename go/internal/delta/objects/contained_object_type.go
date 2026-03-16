package objects

type containedObjectType byte

const (
	containedObjectTypeTag containedObjectType = iota
	containedObjectTypeBlobReferences
	containedObjectTypeReference
)

func (t containedObjectType) IsTag() bool           { return t == containedObjectTypeTag }
func (t containedObjectType) IsBlobReference() bool { return t == containedObjectTypeBlobReferences }
func (t containedObjectType) IsReference() bool     { return t == containedObjectTypeReference }
