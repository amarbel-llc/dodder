package objects

type ContainedObjectType interface {
	containedObjectType()
}

type containedObjectType byte

func (containedObjectType) containedObjectType() {}

const (
	containedObjectTypeMetadataExplicit = iota
	containedObjectTypeBlobReferences
)
