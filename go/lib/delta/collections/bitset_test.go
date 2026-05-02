package collections

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestBitset0CapGreaterAdd(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)
	sut.Add(19)

	if !sut.Get(19) {
		t.Errorf("expected bitset to contain idx %d", 19)
	}
}

func TestBitset1CapLessAdd(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)
	toAdd := int(21)
	sut.Add(toAdd)

	if !sut.Get(toAdd) {
		t.Errorf("expected bitset to contain idx %d", toAdd)
	}
}

func TestBitset2CapLessAddRemove(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)
	toAdd := int(256)
	sut.Add(toAdd)

	if !sut.Get(toAdd) {
		t.Errorf("expected bitset to contain idx %d", toAdd)
	}

	sut.Del(toAdd)

	if sut.Get(toAdd) {
		t.Errorf("expected bitset to not contain idx %d", toAdd)
	}
}

func TestBitset3WouldGrowTooLarge(t1 *testing.T) {
	t := ui.T{T: t1}

	defer func() {
		e := recover()

		if e == nil {
			t.Errorf("expected bitset to panic")
		}
	}()

	sut := MakeBitset(20)
	toAdd := int(MaxBitsetIdx + 1)
	sut.Add(toAdd)
}

func TestBitset5Equals(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)
	toAdd := 12
	sut.Add(toAdd)

	sut2 := MakeBitset(20)
	sut2.Add(toAdd)

	if !sut.Equals(sut2) {
		t.Errorf("expected equality")
	}
}

func TestBitset6MakeOn(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitsetOn(20)

	for i := 0; i < 20; i++ {
		if !sut.Get(i) {
			t.Errorf("expected bit to be on: %d", i)
		}
	}

	if sut.Get(21) {
		t.Errorf("expected bit outside range to be off")
	}
}

func TestBitset7Each(t1 *testing.T) {
	t := ui.T{T: t1}

	m := 200
	sut := MakeBitsetOn(m)

	i := 0

	if err := sut.EachOn(
		func(j int) (err error) {
			if j > m {
				t.Errorf("expected to iterate to %d but only got %d", m, j)
			}

			if j != i {
				t.Errorf("expected %d but got %d", i, j)
			}

			i++

			return err
		},
	); err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	if i != m {
		t.Errorf("expected to iterate to %d but only got %d", m, i)
	}
}

func TestBitsetBinaryRoundTripSingle(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)
	sut.Add(12)

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	sut2 := MakeBitset(0)
	if err := sut2.(*bitset).UnmarshalBinary(bs); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}

	if !sut.Equals(sut2) {
		t.Errorf("expected equality after round-trip")
	}

	if sut2.CountOn() != 1 {
		t.Errorf("expected CountOn=1 but got %d", sut2.CountOn())
	}
}

func TestBitsetBinaryRoundTripMultiple(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(200)
	sut.Add(0)
	sut.Add(31)
	sut.Add(32)
	sut.Add(63)
	sut.Add(100)
	sut.Add(199)

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	sut2 := MakeBitset(0)
	if err := sut2.(*bitset).UnmarshalBinary(bs); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}

	if !sut.Equals(sut2) {
		t.Errorf("expected equality after round-trip")
	}

	if sut2.CountOn() != 6 {
		t.Errorf("expected CountOn=6 but got %d", sut2.CountOn())
	}

	for _, idx := range []int{0, 31, 32, 63, 100, 199} {
		if !sut2.Get(idx) {
			t.Errorf("expected bit %d to be set after round-trip", idx)
		}
	}
}

func TestBitsetBinaryRoundTripEmpty(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(20)

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	sut2 := MakeBitset(0)
	if err := sut2.(*bitset).UnmarshalBinary(bs); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}

	if sut2.CountOn() != 0 {
		t.Errorf("expected CountOn=0 but got %d", sut2.CountOn())
	}
}

func TestBitsetBinaryRoundTripAllOn(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitsetOn(200)

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	sut2 := MakeBitset(0)
	if err := sut2.(*bitset).UnmarshalBinary(bs); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}

	if !sut.Equals(sut2) {
		t.Errorf("expected equality after round-trip")
	}

	if sut2.CountOn() != 200 {
		t.Errorf("expected CountOn=200 but got %d", sut2.CountOn())
	}
}

func TestBitsetBinarySize(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(64)
	sut.Add(0)
	sut.Add(63)

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	// 64 bits = 2 uint32s = 8 bytes
	if len(bs) != 8 {
		t.Errorf("expected 8 bytes for 64-bit bitset, got %d", len(bs))
	}
}

func TestBitsetBinaryCountOnAfterUnmarshal(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(100)
	for i := 0; i < 50; i++ {
		sut.Add(i * 2) // add 50 even-numbered bits
	}

	bs, err := sut.(*bitset).MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %s", err)
	}

	sut2 := MakeBitset(0)
	if err := sut2.(*bitset).UnmarshalBinary(bs); err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}

	if sut2.CountOn() != 50 {
		t.Errorf("expected CountOn=50 but got %d (countOn accumulation bug?)", sut2.CountOn())
	}
}

func TestNthOnBasic(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(200)
	sut.Add(0)
	sut.Add(31)
	sut.Add(32)
	sut.Add(63)
	sut.Add(100)
	sut.Add(199)

	expected := []int{0, 31, 32, 63, 100, 199}
	for i, ex := range expected {
		idx, ok := sut.NthOn(i)
		if !ok {
			t.Errorf("NthOn(%d) returned not found", i)
			continue
		}
		if idx != ex {
			t.Errorf("NthOn(%d) = %d, want %d", i, idx, ex)
		}
	}

	_, ok := sut.NthOn(6)
	if ok {
		t.Errorf("NthOn(6) should return not found for 6-element bitset")
	}
}

func TestNthOnAllOn(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitsetOn(100)

	for i := 0; i < 100; i++ {
		idx, ok := sut.NthOn(i)
		if !ok {
			t.Fatalf("NthOn(%d) returned not found", i)
		}
		if idx != i {
			t.Errorf("NthOn(%d) = %d, want %d", i, idx, i)
		}
	}
}

func TestNthOnEmpty(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeBitset(100)

	_, ok := sut.NthOn(0)
	if ok {
		t.Errorf("NthOn(0) should return not found for empty bitset")
	}
}

func TestNthOnSparse(t1 *testing.T) {
	t := ui.T{T: t1}

	// Set bits at word boundaries to test cross-word skipping
	sut := MakeBitset(1000)
	sut.Add(33)
	sut.Add(500)
	sut.Add(999)

	cases := []struct {
		n    int
		want int
	}{
		{0, 33},
		{1, 500},
		{2, 999},
	}

	for _, c := range cases {
		idx, ok := sut.NthOn(c.n)
		if !ok {
			t.Errorf("NthOn(%d) returned not found", c.n)
			continue
		}
		if idx != c.want {
			t.Errorf("NthOn(%d) = %d, want %d", c.n, idx, c.want)
		}
	}
}

func BenchmarkNthOnPopcount(b *testing.B) {
	sut := MakeBitsetOn(11136) // realistic zettel_id_index size
	// Remove ~half the bits to simulate partially used index
	for i := 0; i < 5000; i += 2 {
		sut.Del(i)
	}

	target := sut.CountOn() / 2

	b.ResetTimer()
	for range b.N {
		sut.NthOn(target)
	}
}

func BenchmarkNthOnVsEachOn(b *testing.B) {
	sut := MakeBitsetOn(11136)
	for i := 0; i < 5000; i += 2 {
		sut.Del(i)
	}

	target := sut.CountOn() / 2

	b.Run("NthOn", func(b *testing.B) {
		for range b.N {
			sut.NthOn(target)
		}
	})

	b.Run("EachOn", func(b *testing.B) {
		for range b.N {
			j := 0
			sut.EachOn(func(n int) error {
				j++
				if j == target {
					return errors.MakeErrStopIteration()
				}
				return nil
			})
		}
	})
}

func BenchmarkAdd(b *testing.B) {
	sut := MakeBitset(int(b.N))

	b.ResetTimer()

	j := int(0)

	for i := 0; i < b.N; i++ {
		if j > MaxBitsetIdx {
			j = 0
		}

		sut.Add(int(j))
		j++
	}
}
