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

type Coordinate struct {
	mazelib.Coordinate
}

func NewCoordinate(x int, y int) Coordinate {
	return Coordinate{mazelib.Coordinate{X: x, Y: y}}
}

func (c Coordinate) Left() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X - 1, c.Y}}
}

func (c Coordinate) Right() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X + 1, c.Y}}
}

func (c Coordinate) Up() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X, c.Y - 1}}
}

func (c Coordinate) Down() Coordinate {
	return Coordinate{mazelib.Coordinate{c.X, c.Y + 1}}
}

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

func (c Coordinate) Neighbors() []Coordinate {
	return []Coordinate{c.Up(), c.Down(), c.Left(), c.Right()}
}

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
