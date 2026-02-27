package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/remote_connection_types"
	"code.linenisgreat.com/dodder/go/internal/echo/xdg"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

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
