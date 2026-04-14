package blob_store_id

import "github.com/amarbel-llc/madder/go/pkgs/xdg_location_type"

type (
	LocationType       = xdg_location_type.Type
	LocationTypeGetter = xdg_location_type.TypeGetter
)

var (
	LocationTypeUnknown   = xdg_location_type.Unknown
	LocationTypeCwd       = xdg_location_type.Cwd
	LocationTypeXDGUser   = xdg_location_type.XDGUser
	LocationTypeXDGSystem = xdg_location_type.XDGSystem
)
