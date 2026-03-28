package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/_/remote_connection_types"
	charlie_rb "code.linenisgreat.com/dodder/go/internal/charlie/repo_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
	"code.linenisgreat.com/dodder/go/lib/echo/xdg"
)

type (
	TomlLocalOverridePathV0 = charlie_rb.TomlLocalOverridePathV0
	TomlXDGV0               = charlie_rb.TomlXDGV0
	TomlUriV0               = charlie_rb.TomlUriV0
)

var TomlXDGV0FromXDG = charlie_rb.TomlXDGV0FromXDG

type (
	Blob interface {
		GetPublicKey() domain_interfaces.MarklId
		IsRemote() bool
	}

	BlobMutable interface {
		Blob
		SetPublicKey(domain_interfaces.MarklId)
	}

	BlobXDG interface {
		Blob
		MakeXDG(utilityName string) xdg.XDG
	}

	BlobOverridePath interface {
		Blob
		GetOverridePath() string
	}

	BlobUri interface {
		Blob
		GetUri() values.Uri
	}
)

var (
	_ BlobOverridePath = TomlLocalOverridePathV0{}
	_ BlobMutable      = &TomlLocalOverridePathV0{}
	_ BlobXDG          = TomlXDGV0{}
	_ BlobMutable      = &TomlXDGV0{}
	_ BlobUri          = TomlUriV0{}
	_ BlobMutable      = &TomlUriV0{}
)

func GetSupportedConnectionTypes(
	blob Blob,
) interfaces.Set[remote_connection_types.Type] {
	if blob.IsRemote() {
		return collections_value.MakeValueSetValue(
			nil,
			remote_connection_types.TypeSocketUnix,
			remote_connection_types.TypeUrl,
			remote_connection_types.TypeStdioSSH,
		)
	} else {
		return collections_value.MakeValueSetValue(
			nil,
			remote_connection_types.TypeNative,
			remote_connection_types.TypeNativeLocalOverridePath,
			remote_connection_types.TypeSocketUnix,
			remote_connection_types.TypeStdioLocal,
		)
	}
}
