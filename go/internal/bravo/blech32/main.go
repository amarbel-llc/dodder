// Copyright (c) 2017 Takatoshi Nakagawa
// Copyright (c) 2019 The age Authors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package blech32 is a modified version bech32 backage which itself is a
// modified version of the reference implementation of BIP173. This package
// changes the spec so that the last occurrence of `1` is actually a `-`
// instead.
package blech32

import (
	"bytes"
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
)

const separator = '-'

var (
	charsetString = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	charset       = []byte(charsetString)
)

var generator = []uint32{
	0x3b6a57b2,
	0x26508e6d,
	0x1ea119fa,
	0x3d4233dd,
	0x2a1462b3,
}

func polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, v := range values {
		top := chk >> 25
		chk = (chk & 0x1ffffff) << 5
		chk = chk ^ uint32(v)
		for i := range 5 {
			bit := top >> i & 1
			if bit == 1 {
				chk ^= generator[i]
			}
		}
	}
	return chk
}

func hrpExpand(hrp string) []byte {
	if hrp == "" {
		return nil
	}

	h := []byte(strings.ToLower(hrp))
	var ret []byte
	for _, c := range h {
		ret = append(ret, c>>5)
	}
	ret = append(ret, 0)
	for _, c := range h {
		ret = append(ret, c&31)
	}
	return ret
}

func verifyChecksum(hrp string, data []byte) bool {
	return polymod(append(hrpExpand(hrp), data...)) == 1
}

func createChecksum(hrp string, data []byte) []byte {
	values := append(hrpExpand(hrp), data...)
	values = append(values, []byte{0, 0, 0, 0, 0, 0}...)
	mod := polymod(values) ^ 1
	ret := make([]byte, 6)
	for p := range ret {
		shift := 5 * (5 - p)
		ret[p] = byte(mod>>shift) & 31
	}
	return ret
}

func convertBits(data []byte, frombits, tobits byte, pad bool) ([]byte, error) {
	var ret []byte
	acc := uint32(0)
	bits := byte(0)
	maxv := byte(1<<tobits - 1)
	for idx, value := range data {
		if value>>frombits != 0 {
			return nil, fmt.Errorf(
				"invalid data range: data[%d]=%d (frombits=%d)",
				idx,
				value,
				frombits,
			)
		}
		acc = acc<<frombits | uint32(value)
		bits += frombits
		for bits >= tobits {
			bits -= tobits
			ret = append(ret, byte(acc>>bits)&maxv)
		}
	}
	if pad {
		if bits > 0 {
			ret = append(ret, byte(acc<<(tobits-bits))&maxv)
		}
	} else if bits >= frombits {
		return nil, fmt.Errorf("illegal zero padding")
	} else if byte(acc<<(tobits-bits))&maxv != 0 {
		return nil, fmt.Errorf("non-zero padding")
	}
	return ret, nil
}

// Encode encodes the HRP and a bytes slice to Blech32. If the HRP is uppercase,
// the output will be uppercase.
func Encode(hrp string, data []byte) ([]byte, error) {
	if len(hrp) < 1 {
		return nil, ErrEmptyHRP
	}
	for p, c := range hrp {
		if c < 33 || c > 126 {
			// TODO turn into error type
			return nil, fmt.Errorf("invalid HRP character: hrp[%d]=%d", p, c)
		}
	}
	return encode(hrp, data)
}

func EncodeDataOnly(data []byte) ([]byte, error) {
	return encode("", data)
}

// Encode encodes the HRP and a bytes slice to Blech32. If the HRP is uppercase,
// the output will be uppercase.
func encode(hrp string, data []byte) ([]byte, error) {
	values, err := convertBits(data, 8, 5, true)
	if err != nil {
		return nil, err
	}

	var lower bool

	if lower, err = validateCaseString(hrp); err != nil {
		return nil, err
	}

	hrp = strings.ToLower(hrp)

	var ret bytes.Buffer

	if hrp != "" {
		ret.WriteString(hrp)
		ret.WriteString("-")
	}

	for _, p := range values {
		ret.WriteByte(charsetString[p])
	}

	for _, p := range createChecksum(hrp, values) {
		ret.WriteByte(charsetString[p])
	}

	if lower {
		return ret.Bytes(), nil
	}

	return bytes.ToUpper(ret.Bytes()), nil
}

func EncodeHRPAsData(hrp string, data []byte) ([]byte, error) {
	dataConverted, err := convertBits(data, 8, 5, true)
	if err != nil {
		return nil, err
	}

	var lower bool

	hrpBytes := []byte(hrp)

	if lower, err = validateCase(hrpBytes); err != nil {
		return nil, err
	}

	unicorn.ToLower(hrpBytes)

	var ret bytes.Buffer

	hrpBytesConverted, err := convertBits(hrpBytes, 8, 5, true)
	if err != nil {
		return nil, err
	}

	for _, p := range hrpBytesConverted {
		ret.WriteByte(charsetString[p])
	}

	for _, p := range dataConverted {
		ret.WriteByte(charsetString[p])
	}

	for _, p := range createChecksum(hrp, dataConverted) {
		ret.WriteByte(charsetString[p])
	}

	if lower {
		return ret.Bytes(), nil
	}

	return bytes.ToUpper(ret.Bytes()), nil
}

func validateHRP(hrp string) (err error) {
	for p, c := range hrp {
		if c < 33 || c > 126 {
			// TODO turn into error type
			return fmt.Errorf(
				"invalid character human-readable part: s[%d]=%d",
				p,
				c,
			)
		}
	}

	return err
}

func validateCaseString(s string) (lower bool, err error) {
	toLower := strings.ToLower(s)
	toUpper := strings.ToUpper(s)

	if toLower != s && toUpper != s {
		// TODO turn into error type
		err = fmt.Errorf("mixed case")
		return lower, err
	} else {
		lower = toLower == s
	}

	return lower, err
}

func validateCase(bites []byte) (lower bool, err error) {
	lowerCount, _, upperCount := unicorn.CountCase(bites)

	if lowerCount != 0 && upperCount != 0 {
		err = fmt.Errorf(
			"mixed case: lower: %d, upper: %d",
			lowerCount,
			upperCount,
		)
		return lower, err
	}

	lower = upperCount == 0

	return lower, err
}

type bytesOrString interface {
	~[]byte | ~string
}

const dataPortionMinWidth = 7

func validateSeparatorPosition[INPUT bytesOrString](
	input INPUT,
	pos int,
) error {
	if pos < 1 {
		return ErrSeparatorMissing
	} else if pos+dataPortionMinWidth > len(input) {
		// TODO turn into error type
		return fmt.Errorf(
			"separator `-` at invalid position because data+checksum portion is too short. Should be at least %d but was %d (%q)",
			dataPortionMinWidth,
			len(input)-(pos+1),
			input[pos+1:],
		)
	}

	return nil
}

// DecodeString decodes a Blech32 string. If the string is uppercase, the HRP
// will be
// uppercase.
func DecodeString(input string) (hrp string, data []byte, err error) {
	if _, err = validateCaseString(input); err != nil {
		return hrp, data, err
	}

	pos := strings.LastIndex(input, "-")

	if err = validateSeparatorPosition(input, pos); err != nil {
		return hrp, data, err
	}

	hrp = input[:pos]

	if err = validateHRP(hrp); err != nil {
		return hrp, data, err
	}

	input = strings.ToLower(input)

	for p, c := range input[pos+1:] {
		d := strings.IndexRune(charsetString, c)
		if d == -1 {
			return "", nil, errInvalidCharacterInData{
				pos:  p + pos + 1,
				char: rune(c),
			}
		}
		data = append(data, byte(d))
	}
	if !verifyChecksum(hrp, data) {
		// TODO turn into error type
		return "", nil, fmt.Errorf("invalid checksum")
	}
	data, err = convertBits(data[:len(data)-6], 5, 8, false)
	if err != nil {
		return "", nil, err
	}
	return hrp, data, nil
}

// Decode decodes a Blech32 string. If the string is uppercase, the HRP
// will be uppercase.
func Decode(bites []byte) (hrp string, data []byte, err error) {
	pos := bytes.LastIndex(bites, []byte("-"))

	if err = validateSeparatorPosition(bites, pos); err != nil {
		return hrp, data, err
	}

	hrp = string(bites[:pos])
	bites = bites[pos+1:]

	if data, err = decode(hrp, bites); err != nil {
		return hrp, data, err
	}

	return hrp, data, err
}

// Decode decodes a Blech32 string. If the string is uppercase, the HRP
// will be uppercase.
func DecodeDataOnly(bites []byte) (data []byte, err error) {
	if data, err = decode("", bites); err != nil {
		return data, err
	}

	return data, err
}

// Decode decodes a Blech32 string. If the string is uppercase, the HRP
// will be uppercase.
func decode(hrp string, bites []byte) (data []byte, err error) {
	if _, err = validateCase(bites); err != nil {
		return data, err
	}

	unicorn.ToLower(bites)

	for p, c := range bites {
		// TODO make more performance with a lookup map
		d := bytes.IndexRune(charset, rune(c))

		if d == -1 {
			return nil, errInvalidCharacterInData{
				pos:  p,
				char: rune(c),
			}
		}

		data = append(data, byte(d))
	}

	if !verifyChecksum(hrp, data) {
		// TODO turn into error type
		return nil, fmt.Errorf("invalid checksum")
	}

	data, err = convertBits(data[:len(data)-6], 5, 8, false)
	if err != nil {
		return nil, err
	}

	return data, nil
}
