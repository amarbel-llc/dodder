package type_blobs

import (
	golf_tb "code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
)

type (
	Blob                   = golf_tb.Blob
	WithFormatters         = golf_tb.WithFormatters
	WithFormatterUTIGroups = golf_tb.WithFormatterUTIGroups
	WithStringLuaHooks     = golf_tb.WithStringLuaHooks
	WithReferences         = golf_tb.WithReferences
	WithFields             = golf_tb.WithFields
	UTIGroup               = golf_tb.UTIGroup
	ReferencesConfig       = golf_tb.ReferencesConfig
	FieldDefinition        = golf_tb.FieldDefinition
	TomlV0                 = golf_tb.TomlV0
	TomlV1                 = golf_tb.TomlV1
	TomlV2                 = golf_tb.TomlV2
)

var (
	Default                    = golf_tb.Default
	DefaultWithPandocFormatter = golf_tb.DefaultWithPandocFormatter
	DefaultPandocDefaults      = golf_tb.DefaultPandocDefaults
	DefaultPandocLuaFilter     = golf_tb.DefaultPandocLuaFilter
	DefaultTaskType            = golf_tb.DefaultTaskType
	DefaultChoreType           = golf_tb.DefaultChoreType
)

var (
	_ Blob = &TomlV0{}
	_ Blob = &TomlV1{}
	_ Blob = &TomlV2{}
)

var (
	DecodeTomlV0           = golf_tb.DecodeTomlV0
	DecodeTomlV1           = golf_tb.DecodeTomlV1
	DecodeTomlV2           = golf_tb.DecodeTomlV2
	DecodeReferencesConfig = golf_tb.DecodeReferencesConfig
)
