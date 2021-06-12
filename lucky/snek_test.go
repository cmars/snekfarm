package lucky

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/cmars/snekfarm/api"
)

func TestNodeCardinal(t *testing.T) {
	c := qt.New(t)
	n := newNode(api.Point{X: 3, Y: 3}, "")
	c.Assert(n, qt.Not(qt.IsNil))
	c.Assert(n.above(), qt.Equals, api.Point{X: 3, Y: 4})
	c.Assert(n.below(), qt.Equals, api.Point{X: 3, Y: 2})
	c.Assert(n.leftOf(), qt.Equals, api.Point{X: 2, Y: 3})
	c.Assert(n.rightOf(), qt.Equals, api.Point{X: 4, Y: 3})
}

func testBoard(s string) *board {
	fields := strings.Split(strings.TrimSpace(s), "\n")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	// Reverse the rows to decode the visual representation
	for i := 0; i < len(fields)/2; i++ {
		j := len(fields) - i - 1
		fields[i], fields[j] = fields[j], fields[i]
	}
	h := len(fields)
	w := len(fields[0])
	b := newBoard(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch fields[y][x] {
			case 's':
				b.set(api.Point{X: x, Y: y}, "snek")
			case 'p':
				b.set(api.Point{X: x, Y: y}, "prey")
			case 'f':
				b.set(api.Point{X: x, Y: y}, "food")
			case 'h':
				b.set(api.Point{X: x, Y: y}, "hazard")
			}
		}
	}
	return b
}

func TestBoard(t *testing.T) {
	c := qt.New(t)
	b := testBoard(`
.......
.fss...
...ss..
.f..s..
....s..
f......
.......`[1:])
	c.Assert(b.get(api.Point{0, 0}), qt.Equals, "")
	c.Assert(b.get(api.Point{0, 1}), qt.Equals, "food")
	c.Assert(b.get(api.Point{4, 2}), qt.Equals, "snek")
}

func TestOrientNoBacktrack(t *testing.T) {
	c := qt.New(t)
	b := testBoard(`
.......
.fss...
...ss..
.f..s..
....s..
f......
.......`[1:])
	n := orient(b, api.Point{2, 5})
	c.Assert(n.up, qt.Not(qt.IsNil))
	c.Assert(n.down, qt.Not(qt.IsNil))
	c.Assert(n.left, qt.Not(qt.IsNil))
	c.Assert(n.right, qt.IsNil)
}

func TestOrientNoAutophagy(t *testing.T) {
	c := qt.New(t)
	b := testBoard(`
.......
.fss...
..sss..
.fsss..
....s..
f......
.......`[1:])
	n := orient(b, api.Point{3, 3})
	c.Assert(n.up, qt.IsNil)
	c.Assert(n.down, qt.Not(qt.IsNil))
	c.Assert(n.left, qt.IsNil)
	c.Assert(n.right, qt.IsNil)
}

func TestOrientDecide(t *testing.T) {
	c := qt.New(t)
	b := testBoard(`
.......
.fss...
...ss..
.f..s..
....s..
f......
.......`[1:])
	n := orient(b, api.Point{2, 5})
	dir := decide(n, 6)
	c.Assert(dir, qt.Equals, "left")
}

func TestWallAvoid(t *testing.T) {
	c := qt.New(t)
	b := testBoard(`
.......
.....f.
.......
.......
s......
s......
s......`[1:])
	n := orient(b, api.Point{0, 0})
	dir := decide(n, 3)
	c.Assert(dir, qt.Equals, "right")
}
