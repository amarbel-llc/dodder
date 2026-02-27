package coordinates

import (
	"fmt"
	"math"
	"strconv"
)

type (
	Int   = uint32
	Float = float32
)

type ZettelIdCoordinate struct {
	Left, Right Int
}

func (p ZettelIdCoordinate) Equals(p1 ZettelIdCoordinate) bool {
	return p.Left == p1.Left && p.Right == p1.Right
}

func Extrema(n Int) ZettelIdCoordinate {
	n1 := Float(n)
	return ZettelIdCoordinate{
		Left:  Int(((n1 - 1) * n1 / 2) + 1),
		Right: Int(n1 * ((n1 + 1) / 2)),
	}
}

func (p *ZettelIdCoordinate) SetCoordinates(left, right string) (err error) {
	l, err := strconv.ParseUint(left, 10, 32)
	if err != nil {
		return err
	}

	r, err := strconv.ParseUint(right, 10, 64)
	if err != nil {
		return err
	}

	p.Left = Int(l)
	p.Right = Int(r)

	return err
}

func (p *ZettelIdCoordinate) Set(id string) (err error) {
	i, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}

	p.SetInt(Int(i))

	return err
}

func (p *ZettelIdCoordinate) SetInt(id Int) {
	n := math.Round(math.Sqrt(float64(id * 2)))
	ext := Extrema(Int(n))

	p.Left = id - ext.Left
	p.Right = ext.Right - id
}

func (p ZettelIdCoordinate) Id() Int {
	n := p.Left + p.Right + 1
	ext := Extrema(n)
	return ext.Left + p.Left
}

func (p ZettelIdCoordinate) String() string {
	return fmt.Sprintf("%d/%d: %d", p.Left, p.Right, p.Id())
}
