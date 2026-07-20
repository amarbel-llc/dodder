package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/0/remote_connection_types"
	charlie_rb "code.linenisgreat.com/dodder/go/internal/alfa/repo_blobs"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/collections_value"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/values"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/xdg"
)

type (
	TomlLocalOverridePathV0 = charlie_rb.TomlLocalOverridePathV0
	TomlXDGV0               = charlie_rb.TomlXDGV0
	TomlUriV0               = charlie_rb.TomlUriV0
)

var TomlXDGV0FromXDG = charlie_rb.TomlXDGV0FromXDG

type (
	Blob interface {
		GetPublicKey() mad_domain_interfaces.MarklId
		IsRemote() bool
	}

	BlobMutable interface {
		Blob
		SetPublicKey(mad_domain_interfaces.MarklId)
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
