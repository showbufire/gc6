package common

import "github.com/golangchallenge/gc6/mazelib"

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
