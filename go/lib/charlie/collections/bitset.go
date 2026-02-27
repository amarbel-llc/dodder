package collections

import (
	"encoding/binary"
	"math/bits"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

const (
	MaxBitsetIdx = 100_000
)

type Bitset interface {
	Equals(Bitset) bool
	Len() int
	Cap() int
	Get(int) bool
	CountOn() int
	CountOff() int
	EachOn(interfaces.FuncIter[int]) error
	EachOff(interfaces.FuncIter[int]) error

	Add(int)
	Del(int)
	DelIfPresent(int)
	set(int, bool)
}

const (
	// For compatibility with 32 bit systems
	// storageInt = uint32
	intSize     = 32
	bytesPerInt = intSize / 4
)

// TODO-P4 consider using ranges
type bitset struct {
	slice   []uint32
	countOn int
	lock    *sync.Mutex
}

func MakeBitset(n int) Bitset {
	return makeBitset(n)
}

func MakeBitsetOn(n int) Bitset {
	b := makeBitset(n)

	if n == 0 {
		return b
	}

	for i := range b.slice {
		b.slice[i] = ^uint32(0)
	}

	last := n / intSize
	lastBitsOn := (n % intSize)

	b.slice[last] = 0

	for i := 0; i < lastBitsOn; i++ {
		b.slice[last] |= (uint32(1) << i)
	}

	return b
}

func makeBitset(n int) (bs *bitset) {
	bs = &bitset{
		slice: make([]uint32, (n+intSize-1)/intSize),
		lock:  &sync.Mutex{},
	}

	return bs
}

func (b bitset) Cap() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return int(intSize * cap(b.slice))
}

func (b bitset) len() int {
	return int(intSize * len(b.slice))
}

func (b bitset) Len() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.len()
}

func (b bitset) CountOn() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.countOn
}

func (b bitset) CountOff() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.len() - b.countOn
}

func (b bitset) get(idx int) bool {
	pos := idx / intSize
	j := uint32(idx % intSize)
	return (b.slice[pos] & (uint32(1) << j)) != 0
}

func (b bitset) Get(idx int) bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.get(idx)
}

func (b bitset) EachOff(f interfaces.FuncIter[int]) (err error) {
	ui.TodoP4("measure and improve performance if necessary")

	b.lock.Lock()
	defer b.lock.Unlock()

	for i := 0; i < b.len(); i++ {
		if b.get(i) {
			continue
		}

		if err = f(i); err != nil {
			if errors.IsStopIteration(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}
	}

	return err
}

func (b bitset) EachOn(f interfaces.FuncIter[int]) (err error) {
	ui.TodoP4("measure and improve performance if necessary")

	b.lock.Lock()
	defer b.lock.Unlock()

	for i := 0; i < b.len(); i++ {
		if !b.get(i) {
			continue
		}

		if err = f(i); err != nil {
			if errors.IsStopIteration(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}
	}

	return err
}

func (a bitset) Equals(b Bitset) bool {
	a.lock.Lock()
	defer a.lock.Unlock()

	ui.TodoP4("should lock b beyond len call")
	if bb, ok := b.(*bitset); ok {
		bb.lock.Lock()
		defer bb.lock.Unlock()

		if bb.len() != a.len() {
			return false
		}

		for i, av := range a.slice {
			if bb.slice[i] != av {
				return false
			}
		}
	} else {
		ui.TodoP4("improve performance of this")
		if bb.Len() != a.len() {
			return false
		}

		for i := 0; i < a.Len(); i++ {
			if a.Get(i) != b.Get(i) {
				return false
			}
		}
	}

	return true
}

func (b *bitset) Add(idx int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.countOn += 1
	b.set(idx, true)
}

func (b *bitset) Del(idx int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.countOn -= 1
	b.set(idx, false)
}

func (b *bitset) DelIfPresent(idx int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.len() < idx {
		return
	}

	if !b.get(idx) {
		return
	}

	b.countOn -= 1
	b.set(idx, false)
}

func (b *bitset) growIfNecessary(idx int) {
	if b.len() > idx {
		return
	}

	newSize := int(float32(idx) * float32(1.5))

	if newSize > MaxBitsetIdx {
		panic("would grow too large")
	}

	a := makeBitset(int(float32(idx) * float32(1.5)))
	copy(a.slice, b.slice)
	b.slice = a.slice

	return
}

func (b *bitset) set(idx int, value bool) {
	b.growIfNecessary(idx)

	pos := idx / intSize
	j := uint32(idx % intSize)

	if value {
		b.slice[pos] |= (uint32(1) << j)
	} else {
		b.slice[pos] &= ^(uint32(1) << j)
	}
}

func (b bitset) MarshalBinary() (bs []byte, err error) {
	bs = make([]byte, len(b.slice)*bytesPerInt)

	for i, v := range b.slice {
		binary.BigEndian.PutUint32(bs[bytesPerInt*i:], v)
	}

	return bs, err
}

func (b *bitset) UnmarshalBinary(bs []byte) (err error) {
	b.slice = make([]uint32, len(bs)/bytesPerInt)
	b.lock = &sync.Mutex{}

	for i := range b.slice {
		n := binary.BigEndian.Uint32(bs[bytesPerInt*i:])
		b.countOn = bits.OnesCount32(n)
		b.slice[i] = n
	}

	return err
}
