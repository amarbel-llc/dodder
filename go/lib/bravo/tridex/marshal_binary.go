package tridex

import (
	"encoding/binary"
	"slices"
	"sort"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func (tridex *Tridex) MarshalBinary() (bs []byte, err error) {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	all := slices.Collect(tridex.Root.allWithAcc(""))
	sort.Strings(all)

	size := 4
	for _, s := range all {
		size += 4 + len(s)
	}

	bs = make([]byte, 0, size)
	bs = binary.BigEndian.AppendUint32(bs, uint32(len(all)))

	for _, s := range all {
		bs = binary.BigEndian.AppendUint32(bs, uint32(len(s)))
		bs = append(bs, s...)
	}

	return bs, err
}

func (tridex *Tridex) UnmarshalBinary(bs []byte) (err error) {
	tridex.lock.Lock()
	defer tridex.lock.Unlock()

	if len(bs) < 4 {
		if len(bs) == 0 {
			return nil
		}

		err = errors.Errorf("tridex binary too short: %d bytes", len(bs))
		return err
	}

	count := binary.BigEndian.Uint32(bs[:4])
	offset := 4

	tridex.Root = node{
		Children: make(map[byte]node),
		IsRoot:   true,
	}

	for i := uint32(0); i < count; i++ {
		if offset+4 > len(bs) {
			err = errors.Errorf("tridex binary truncated at entry %d", i)
			return err
		}

		sLen := int(binary.BigEndian.Uint32(bs[offset : offset+4]))
		offset += 4

		if offset+sLen > len(bs) {
			err = errors.Errorf("tridex binary truncated at string %d", i)
			return err
		}

		s := string(bs[offset : offset+sLen])
		offset += sLen

		tridex.Root.Add(s)
	}

	return err
}
