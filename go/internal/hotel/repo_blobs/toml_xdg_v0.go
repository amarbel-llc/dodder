package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/echo/xdg"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/delta/xdg_defaults"
)

type TomlXDGV0 struct {
	PublicKey markl.Id `toml:"public-key"`
	Data      string   `toml:"data"`
	Config    string   `toml:"config"`
	State     string   `toml:"state"`
	Cache     string   `toml:"cache"`
	Runtime   string   `toml:"runtime"`
}

var (
	_ BlobXDG     = TomlXDGV0{}
	_ BlobMutable = &TomlXDGV0{}
)

func TomlXDGV0FromXDG(xdg xdg.XDG) *TomlXDGV0 {
	return &TomlXDGV0{
		Data:    xdg.Data.String(),
		Config:  xdg.Config.String(),
		State:   xdg.State.String(),
		Cache:   xdg.Cache.String(),
		Runtime: xdg.Runtime.String(),
	}
}

func (blob TomlXDGV0) MakeXDG(utilityName string) xdg.XDG {
	// TODO fix this init to be correct
	return xdg.XDG{
		UtilityName: utilityName,
		Data:        xdg_defaults.Data.MakeBaseEnvVar(blob.Data),
		Config:      xdg_defaults.Config.MakeBaseEnvVar(blob.Config),
		Cache:       xdg_defaults.Cache.MakeBaseEnvVar(blob.Cache),
		Runtime:     xdg_defaults.Runtime.MakeBaseEnvVar(blob.Runtime),
		State:       xdg_defaults.State.MakeBaseEnvVar(blob.State),
	}
}

func (blob TomlXDGV0) GetRepoBlob() Blob {
	return blob
}

func (blob TomlXDGV0) GetPublicKey() domain_interfaces.MarklId {
	return blob.PublicKey
}

func (blob *TomlXDGV0) SetPublicKey(id domain_interfaces.MarklId) {
	blob.PublicKey.ResetWithMarklId(id)
}

func (blob TomlXDGV0) IsRemote() bool {
	return false
}
