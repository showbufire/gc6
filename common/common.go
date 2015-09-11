// common provides convient abstraction for both client and server.
package common

import "github.com/showbufire/gc6/mazelib"

var (
	ReverseDirection = map[int]int{
		mazelib.N: mazelib.S,
		mazelib.S: mazelib.N,
		mazelib.E: mazelib.W,
		mazelib.W: mazelib.E,
	}
)

// Coordinate embeds the default one
type Coordinate struct {
	mazelib.Coordinate
}

// NewCoordinate creates a Coordinate from x and y
func NewCoordinate(x int, y int) Coordinate {
	return Coordinate{mazelib.Coordinate{X: x, Y: y}}
}

// Left returns the left neighhbor
func (c Coordinate) Left() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X - 1, c.Y}}
}

// Right returns the right neighhbor
func (c Coordinate) Right() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X + 1, c.Y}}
}

// Up returns the top neighbor
func (c Coordinate) Up() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X, c.Y - 1}}
}

// Down returns the bottom neighbor
func (c Coordinate) Down() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X, c.Y + 1}}
}

// Neighbor returns the neighbor at direction dir
func (c Coordinate) Neighbor(dir int) Coordinate {
	var ret Coordinate
	switch dir {
	case mazelib.N:
		ret = c.Up()
	case mazelib.S:
		ret = c.Down()
	case mazelib.W:
		ret = c.Left()
	case mazelib.E:
		ret = c.Right()
	}
	return ret
}

// Neighbors return all the neighbors
func (c Coordinate) Neighbors() []Coordinate {
	return []Coordinate{c.Up(), c.Down(), c.Left(), c.Right()}
}

// GetDir given another Coordinate, which must be a neighbor, returns the direction
func (c Coordinate) GetDir(cc Coordinate) int {
	if c.X == cc.X {
		if c.Y < cc.Y {
			return mazelib.S
		} else {
			return mazelib.N
		}
	}
	if c.X < cc.X {
		return mazelib.E
	}
	return mazelib.W
}
