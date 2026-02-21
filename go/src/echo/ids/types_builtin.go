package ids

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
)

const (
	// TODO figure out a more ergonomic way of incrementing and labeling as
	// latest -> interface {
	//   All() interface.Seq[id.Type]
	// 	 GetCurrent() Type
	// }

	// TODO remove deprecated
	// keep sorted
	TypeInventoryListJsonV0 = "!inventory_list-json-v0"
	TypeInventoryListV0     = "!inventory_list-v0" // Deprevated
	TypeInventoryListV1     = "!inventory_list-v1"
	TypeInventoryListV2     = "!inventory_list-v2"

	TypeLuaTagV1                                    = "!lua-tag-v1" // Deprecated
	TypeLuaTagV2                                    = "!lua-tag-v2"
	TypeTomlBlobStoreConfigSftpExplicitV0           = "!toml-blob_store_config_sftp-explicit-v0"
	TypeTomlBlobStoreConfigSftpViaSSHConfigV0       = "!toml-blob_store_config_sftp-ssh_config-v0"
	TypeTomlBlobStoreConfigV0                       = "!toml-blob_store_config-v0"
	TypeTomlBlobStoreConfigV1                       = "!toml-blob_store_config-v1"
	TypeTomlBlobStoreConfigV2                       = "!toml-blob_store_config-v2"
	TypeTomlBlobStoreConfigPointerV0                = "!toml-blob_store_config-pointer-v0"
	TypeTomlBlobStoreConfigInventoryArchiveV0       = "!toml-blob_store_config-inventory_archive-v0"
	TypeTomlBlobStoreConfigInventoryArchiveV1       = "!toml-blob_store_config-inventory_archive-v1"
	TypeTomlBlobStoreConfigInventoryArchiveVCurrent = TypeTomlBlobStoreConfigInventoryArchiveV1
	TypeTomlBlobStoreConfigVCurrent                 = TypeTomlBlobStoreConfigV2
	TypeTomlConfigImmutableV1                       = "!toml-config-immutable-v1" // Deprecated
	TypeTomlConfigImmutableV2                       = "!toml-config-immutable-v2"
	TypeTomlConfigV0                                = "!toml-config-v0" // Deprecated
	TypeTomlConfigV1                                = "!toml-config-v1"
	TypeTomlConfigV2                                = "!toml-config-v2"
	TypeTomlRepoDotenvXdgV0                         = "!toml-repo-dotenv_xdg-v0"
	TypeTomlRepoLocalOverridePath                   = "!toml-repo-local_override_path-v0"
	TypeTomlRepoUri                                 = "!toml-repo-uri-v0"
	TypeTomlTagV0                                   = "!toml-tag-v0" // Deprecated
	TypeTomlTagV1                                   = "!toml-tag-v1"
	TypeTomlTypeV0                                  = "!toml-type-v0" // Deprecated
	TypeTomlTypeV1                                  = "!toml-type-v1"
	TypeTomlTypeVCurrent                            = TypeTomlTypeV1
	TypeTomlWorkspaceConfigV0                       = "!toml-workspace_config-v0"
	TypeTomlWorkspaceConfigVCurrent                 = TypeTomlWorkspaceConfigV0
	TypeZettelIdListV0                              = "!zettel_id_list-v0" // not used yet

	// Aliases
	TypeInventoryListVCurrent = TypeInventoryListV2
)

type BuiltinType struct {
	TypeStruct
	genres.Genre
	Default bool
}

var (
	allSlice []BuiltinType
	allMap   map[TypeStruct]BuiltinType
	defaults map[genres.Genre]BuiltinType
)

func init() {
	allMap = make(map[TypeStruct]BuiltinType)
	defaults = make(map[genres.Genre]BuiltinType)

	// keep sorted
	registerBuiltinTypeString(TypeInventoryListV0, genres.InventoryList, false)
	registerBuiltinTypeString(TypeInventoryListV1, genres.InventoryList, false)
	registerBuiltinTypeString(TypeInventoryListV2, genres.InventoryList, true)
	registerBuiltinTypeString(
		TypeInventoryListJsonV0,
		genres.InventoryList,
		false,
	)
	registerBuiltinTypeString(TypeLuaTagV1, genres.Tag, false)
	registerBuiltinTypeString(TypeLuaTagV2, genres.Tag, false)
	registerBuiltinTypeString(TypeTomlBlobStoreConfigV0, genres.Unknown, false)
	registerBuiltinTypeString(TypeTomlBlobStoreConfigV1, genres.Unknown, false)
	registerBuiltinTypeString(TypeTomlBlobStoreConfigV2, genres.Unknown, false)
	registerBuiltinTypeString(
		TypeTomlBlobStoreConfigPointerV0,
		genres.Unknown,
		false,
	)
	registerBuiltinTypeString(
		TypeTomlBlobStoreConfigInventoryArchiveV0,
		genres.Unknown,
		false,
	)
	registerBuiltinTypeString(
		TypeTomlBlobStoreConfigInventoryArchiveV1,
		genres.Unknown,
		false,
	)
	registerBuiltinTypeString(
		TypeTomlBlobStoreConfigSftpExplicitV0,
		genres.Unknown,
		false,
	)
	registerBuiltinTypeString(
		TypeTomlBlobStoreConfigSftpViaSSHConfigV0,
		genres.Unknown,
		false,
	)
	registerBuiltinTypeString(TypeTomlConfigImmutableV1, genres.Unknown, false)
	registerBuiltinTypeString(TypeTomlConfigImmutableV2, genres.Unknown, false)
	registerBuiltinTypeString(TypeTomlConfigV0, genres.Config, false)
	registerBuiltinTypeString(TypeTomlConfigV1, genres.Config, false)
	registerBuiltinTypeString(TypeTomlConfigV2, genres.Config, true)
	registerBuiltinTypeString(TypeTomlRepoDotenvXdgV0, genres.Repo, false)
	registerBuiltinTypeString(TypeTomlRepoLocalOverridePath, genres.Repo, false)
	registerBuiltinTypeString(TypeTomlRepoUri, genres.Repo, true)
	registerBuiltinTypeString(TypeTomlTagV0, genres.Tag, false)
	registerBuiltinTypeString(TypeTomlTagV1, genres.Tag, true)
	registerBuiltinTypeString(TypeTomlTypeV0, genres.Type, false)
	registerBuiltinTypeString(TypeTomlTypeV1, genres.Type, true)
	registerBuiltinTypeString(TypeTomlWorkspaceConfigV0, genres.Unknown, false)
	registerBuiltinTypeString(TypeZettelIdListV0, genres.Unknown, false)
}

// TODO switch to isDefault being a StoreVersion
func registerBuiltinTypeString(
	tipeString string,
	genre genres.Genre,
	isDefault bool,
) {
	registerBuiltinType(
		BuiltinType{
			TypeStruct: MustTypeStruct(tipeString),
			Genre:      genre,
			Default:    isDefault,
		},
	)
}

func registerBuiltinType(bt BuiltinType) {
	if _, exists := allMap[bt.TypeStruct]; exists {
		panic(
			fmt.Sprintf("builtin type registered more than once: %s", bt.TypeStruct),
		)
	}

	if _, exists := defaults[bt.Genre]; exists && bt.Default {
		panic(
			fmt.Sprintf(
				"builtin default type registered more than once: %s",
				bt.TypeStruct,
			),
		)
	}

	allMap[bt.TypeStruct] = bt
	allSlice = append(allSlice, bt)

	if bt.Default {
		defaults[bt.Genre] = bt
	}
}

func ObjectIdToTypeStruct(id domain_interfaces.ObjectId) TypeStruct {
	var tipe TypeStruct

	switch id := id.(type) {
	default:
		panic(fmt.Sprintf("not a type: %T", id))

	case SeqId:
		tipe = id.ToType()

	case *SeqId:
		tipe = id.ToType()

	case TypeStruct:
		tipe = id

	case *TypeStruct:
		tipe = *id
	}

	return tipe
}

func IsBuiltin(id domain_interfaces.ObjectId) bool {
	tipe := ObjectIdToTypeStruct(id)
	_, ok := allMap[tipe]
	return ok
}

func Get(id domain_interfaces.ObjectId) (BuiltinType, bool) {
	tipe := ObjectIdToTypeStruct(id)
	bt, ok := allMap[tipe]
	return bt, ok
}

func GetOrPanic(idString string) BuiltinType {
	tipe := MustTypeStruct(idString)
	bt, ok := Get(tipe)

	if !ok {
		panic(fmt.Sprintf("no builtin type found for %q", tipe))
	}

	return bt
}

func Default(genre genres.Genre) (TypeStruct, bool) {
	bt, ok := defaults[genre]
	return bt.TypeStruct, ok
}

func DefaultOrPanic(genre genres.Genre) TypeStruct {
	t, ok := Default(genre)

	if !ok {
		panic(fmt.Sprintf("default missing for genre %q", genre))
	}

	return t
}
