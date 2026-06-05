package zettel_id_provider

import (
	"bytes"
	_ "embed"
	"io"
)

// Default zettel-id word lists, used by `dodder init-default` (and the
// `-yin-default` / `-yang-default` opt-in flags on `dodder init`) so a
// repository can be created with a usable zettel-id vocabulary without
// the caller supplying yin/yang files. One lowercased word per line;
// the genesis ingestion runs each through Clean and dedups.

//go:embed embedded/yin.txt
var defaultYin []byte

//go:embed embedded/yang.txt
var defaultYang []byte

// DefaultYinReader returns a reader over the embedded default yin (left
// part) word list.
func DefaultYinReader() io.Reader {
	return bytes.NewReader(defaultYin)
}

// DefaultYangReader returns a reader over the embedded default yang
// (right part) word list.
func DefaultYangReader() io.Reader {
	return bytes.NewReader(defaultYang)
}
