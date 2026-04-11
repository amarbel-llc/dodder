# bech32

BIP173 Bech32 encoding/decoding implementation (modified from reference).

## Key Functions

- `Encode(hrp string, data []byte)` - encodes HRP and bytes to Bech32 string
- `Decode(s string)` - decodes Bech32 string to HRP and data bytes
- `convertBits()` - converts between bit groupings (8<->5 bits)

## Notes

- Supports both uppercase and lowercase output
- Includes checksum verification
- Originally from github.com/takatoshi-nakagawa/age project
