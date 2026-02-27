package cmp

import (
	"bytes"
	"cmp"
)

func Bytes(left, right []byte) Result {
	cmp := bytes.Compare(left, right)
	return result(cmp)
}

func Ordered[ELEMENT cmp.Ordered](left, right ELEMENT) Result {
	if left < right {
		return Less
	} else if left == right {
		return Equal
	} else {
		return Greater
	}
}

func String(left, right string) Result {
	if left < right {
		return Less
	} else if left == right {
		return Equal
	} else {
		return Greater
	}
}
