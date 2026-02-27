package toml

import "github.com/pelletier/go-toml/v2"

var (
	Unmarshal  = toml.Unmarshal
	Marshal    = toml.Marshal
	NewDecoder = toml.NewDecoder
	NewEncoder = toml.NewEncoder
)
