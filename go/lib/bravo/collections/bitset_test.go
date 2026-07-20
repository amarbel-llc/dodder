package collections

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestBitset0CapGreaterAdd(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)
	sut.Add(19)

	t.AssertTrue(sut.Get(19), "expected bitset to contain idx 19")
}

func TestBitset1CapLessAdd(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)
	toAdd := int(21)
	sut.Add(toAdd)

	t.AssertTrue(sut.Get(toAdd), "expected bitset to contain idx")
}

func TestBitset2CapLessAddRemove(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)
	toAdd := int(256)
	sut.Add(toAdd)

	t.AssertTrue(sut.Get(toAdd), "expected bitset to contain idx")

	sut.Del(toAdd)

	t.AssertFalse(sut.Get(toAdd), "expected bitset to not contain idx")
}

func TestBitset3WouldGrowTooLarge(t1 *testing.T) {
	t := ui.MakeT(t1)

	t.AssertPanic(func() {
		sut := MakeBitset(20)
		toAdd := int(MaxBitsetIdx + 1)
		sut.Add(toAdd)
	})
}

func TestBitset5Equals(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)
	toAdd := 12
	sut.Add(toAdd)

	sut2 := MakeBitset(20)
	sut2.Add(toAdd)

	t.AssertTrue(sut.Equals(sut2), "expected equality")
}

func TestBitset6MakeOn(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitsetOn(20)

	for i := 0; i < 20; i++ {
		t.AssertTrue(sut.Get(i), "expected bit to be on")
	}

	t.AssertFalse(sut.Get(21), "expected bit outside range to be off")
}

func TestBitset7Each(t1 *testing.T) {
	t := ui.MakeT(t1)

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

	t.AssertEqual(m, i)
}

func TestBitsetBinaryRoundTripSingle(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)
	sut.Add(12)

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	sut2 := MakeBitset(0)
	t.AssertNoError(sut2.(*bitset).UnmarshalBinary(bs))

	t.AssertTrue(sut.Equals(sut2), "expected equality after round-trip")

	t.AssertEqual(1, sut2.CountOn())
}

func TestBitsetBinaryRoundTripMultiple(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(200)
	sut.Add(0)
	sut.Add(31)
	sut.Add(32)
	sut.Add(63)
	sut.Add(100)
	sut.Add(199)

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	sut2 := MakeBitset(0)
	t.AssertNoError(sut2.(*bitset).UnmarshalBinary(bs))

	t.AssertTrue(sut.Equals(sut2), "expected equality after round-trip")

	t.AssertEqual(6, sut2.CountOn())

	for _, idx := range []int{0, 31, 32, 63, 100, 199} {
		t.AssertTrue(sut2.Get(idx), "expected bit to be set after round-trip")
	}
}

func TestBitsetBinaryRoundTripEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(20)

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	sut2 := MakeBitset(0)
	t.AssertNoError(sut2.(*bitset).UnmarshalBinary(bs))

	t.AssertEqual(0, sut2.CountOn())
}

func TestBitsetBinaryRoundTripAllOn(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitsetOn(200)

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	sut2 := MakeBitset(0)
	t.AssertNoError(sut2.(*bitset).UnmarshalBinary(bs))

	t.AssertTrue(sut.Equals(sut2), "expected equality after round-trip")

	t.AssertEqual(200, sut2.CountOn())
}

func TestBitsetBinarySize(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(64)
	sut.Add(0)
	sut.Add(63)

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	// 64 bits = 2 uint32s = 8 bytes
	t.AssertEqual(8, len(bs))
}

func TestBitsetBinaryCountOnAfterUnmarshal(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(100)
	for i := 0; i < 50; i++ {
		sut.Add(i * 2) // add 50 even-numbered bits
	}

	bs, err := sut.(*bitset).MarshalBinary()
	t.AssertNoError(err)

	sut2 := MakeBitset(0)
	t.AssertNoError(sut2.(*bitset).UnmarshalBinary(bs))

	t.AssertEqual(50, sut2.CountOn())
}

func TestNthOnBasic(t1 *testing.T) {
	t := ui.MakeT(t1)

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
		t.AssertEqual(ex, idx)
	}

	_, ok := sut.NthOn(6)
	t.AssertFalse(ok, "NthOn(6) should return not found for 6-element bitset")
}

func TestNthOnAllOn(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitsetOn(100)

	for i := 0; i < 100; i++ {
		idx, ok := sut.NthOn(i)
		if !ok {
			t.Fatalf("NthOn(%d) returned not found", i)
		}
		t.AssertEqual(i, idx)
	}
}

func TestNthOnEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := MakeBitset(100)

	_, ok := sut.NthOn(0)
	t.AssertFalse(ok, "NthOn(0) should return not found for empty bitset")
}

func TestNthOnSparse(t1 *testing.T) {
	t := ui.MakeT(t1)

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
		t.AssertEqual(c.want, idx)
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
