// Package hyphence is dodder's facade over the external hyphence format
// library (github.com/amarbel-llc/hyphence/go/hyphence), pre-filling madder's
// TypeStruct / markl.Id into the library's four leading generic type
// parameters (T, PT, D, PD). dodder's typed blobs are always keyed by a
// madder TypeStruct and a madder markl.Id digest, so this exposes the same
// one-parameter API (`TypedBlob[BLOB]`, `CoderToTypedBlob[BLOB]`, …) that
// madder's now-deleted go/pkgs/hyphence facade provided.
//
// madder extracted hyphence into a standalone repo and dropped its pre-filled
// facade in madder go/v0.4.0 (madder #253). Recreating the facade here keeps
// dodder's ~26 consumer call sites unchanged — they only swap the import path
// from github.com/amarbel-llc/madder/go/pkgs/hyphence to this package.
//
// Generic type aliases require Go 1.24+ (dodder is on 1.26).
package hyphence

import (
	ext "github.com/amarbel-llc/hyphence/go/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// Five-parameter generics, pre-filled with
// T=mad_ids.TypeStruct, PT=*mad_ids.TypeStruct, D=markl.Id, PD=*markl.Id.
type (
	TypedBlob[BLOB any]                 = ext.TypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
	CoderToTypedBlob[BLOB any]          = ext.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
	TypedMetadataCoder[BLOB any]        = ext.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
	CoderTypeMapWithoutType[BLOB any]   = ext.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
	EncoderTypeMapWithoutType[BLOB any] = ext.EncoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
	DecoderTypeMapWithoutType[BLOB any] = ext.DecoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB]
)

// TypedBlobEmpty is a header-only typed blob (no blob payload, BLOB=struct{}).
type TypedBlobEmpty = ext.TypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, struct{}]

// Single-BLOB-parameter generics — the library does not parameterize these
// by the type/digest types, so they pass through unchanged.
type (
	Coder[BLOB any]   = ext.Coder[BLOB]
	Encoder[BLOB any] = ext.Encoder[BLOB]
	Decoder[BLOB any] = ext.Decoder[BLOB]
)

// BLOB + pointer coders, passed through unchanged.
type (
	CoderTommy[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]]       = ext.CoderTommy[BLOB, BLOB_PTR]
	TommyBlobDecoder[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]] = ext.TommyBlobDecoder[BLOB, BLOB_PTR]
	TommyBlobEncoder[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]] = ext.TommyBlobEncoder[BLOB, BLOB_PTR]
)

// Non-generic format types.
type (
	MetadataWriterTo = ext.MetadataWriterTo
	Reader           = ext.Reader
	Writer           = ext.Writer
)

const Boundary = ext.Boundary

// Generic function wrappers — Go cannot alias generic functions
// (golang/go#52654), so these re-fix the four leading type params and
// forward to the external library. BLOB_PTR is inferred from its
// interfaces.Ptr[BLOB] constraint's core type, as it was through madder's
// deleted facade.
func EncodeToFile[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]](
	coders CoderToTypedBlob[BLOB],
	typedBlob *TypedBlob[BLOB],
	path string,
) error {
	return ext.EncodeToFile[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB, BLOB_PTR](
		coders, typedBlob, path,
	)
}

func DecodeFromFile[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]](
	coders CoderToTypedBlob[BLOB],
	path string,
) (TypedBlob[BLOB], error) {
	return ext.DecodeFromFile[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB, BLOB_PTR](
		coders, path,
	)
}

func DecodeFromFileInto[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]](
	typedBlob *TypedBlob[BLOB],
	coders CoderToTypedBlob[BLOB],
	path string,
) error {
	return ext.DecodeFromFileInto[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB, BLOB_PTR](
		typedBlob, coders, path,
	)
}

func DecodeFromFileOrEmptyBuffer[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]](
	coders CoderToTypedBlob[BLOB],
	path string,
	permitNotExist bool,
) (TypedBlob[BLOB], error) {
	return ext.DecodeFromFileOrEmptyBuffer[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, BLOB, BLOB_PTR](
		coders, path, permitNotExist,
	)
}
