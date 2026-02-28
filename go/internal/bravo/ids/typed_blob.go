package ids

type TypedBlob[T any] struct {
	Type TypeStruct
	Blob T
}
