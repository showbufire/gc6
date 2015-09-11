// Copyright © 2015 Steve Francia <spf@spf13.com>.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//

package commands

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/showbufire/gc6/mazelib"
	"github.com/showbufire/gc6/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Maze struct {
	rooms      [][]mazelib.Room
	start      mazelib.Coordinate
	end        mazelib.Coordinate
	icarus     mazelib.Coordinate
	StepsTaken int
}

// Tracking the current maze being solved

// WARNING: This approach is not safe for concurrent use
// This server is only intended to have a single client at a time
// We would need a different and more complex approach if we wanted
// concurrent connections than these simple package variables
var currentMaze *Maze
var scores []int

const (
	cutLimit = 3
)

// Defining the daedalus command.
// This will be called as 'laybrinth daedalus'
var daedalusCmd = &cobra.Command{
	Use:     "daedalus",
	Aliases: []string{"deadalus", "server"},
	Short:   "Start the laybrinth creator",
	Long: `Daedalus's job is to create a challenging Labyrinth for his opponent
  Icarus to solve.

  Daedalus runs a server which Icarus clients can connect to to solve laybrinths.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunServer()
	},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // need to initialize the seed
	gin.SetMode(gin.ReleaseMode)

	RootCmd.AddCommand(daedalusCmd)
}

// Runs the web server
func RunServer() {
	// Adding handling so that even when ctrl+c is pressed we still print
	// out the results prior to exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		printResults()
		os.Exit(1)
	}()

	// Using gin-gonic/gin to handle our routing
	r := gin.Default()
	v1 := r.Group("/")
	{
		v1.GET("/awake", GetStartingPoint)
		v1.GET("/move/:direction", MoveDirection)
		v1.GET("/done", End)
	}

	r.Run(":" + viper.GetString("port"))
}

// Ends a session and prints the results.
// Called by Icarus when he has reached
//   the number of times he wants to solve the laybrinth.
func End(c *gin.Context) {
	printResults()
	os.Exit(1)
}

// initializes a new maze and places Icarus in his awakening location
func GetStartingPoint(c *gin.Context) {
	initializeMaze()
	startRoom, err := currentMaze.Discover(currentMaze.Icarus())
	if err != nil {
		fmt.Println("Icarus is outside of the maze. This shouldn't ever happen")
		fmt.Println(err)
		os.Exit(-1)
	}
	mazelib.PrintMaze(currentMaze)

	c.JSON(http.StatusOK, mazelib.Reply{Survey: startRoom})
}

// The API response to the /move/:direction address
func MoveDirection(c *gin.Context) {
	var err error

	switch c.Param("direction") {
	case "left":
		err = currentMaze.MoveLeft()
	case "right":
		err = currentMaze.MoveRight()
	case "down":
		err = currentMaze.MoveDown()
	case "up":
		err = currentMaze.MoveUp()
	}

	var r mazelib.Reply

	if err != nil {
		r.Error = true
		r.Message = err.Error()
		c.JSON(409, r)
		return
	}

	s, e := currentMaze.LookAround()

	if e != nil {
		if e == mazelib.ErrVictory {
			scores = append(scores, currentMaze.StepsTaken)
			r.Victory = true
			r.Message = fmt.Sprintf("Victory achieved in %d steps \n", currentMaze.StepsTaken)
		} else {
			r.Error = true
			r.Message = err.Error()
		}
	}

	r.Survey = s

	c.JSON(http.StatusOK, r)
}

func initializeMaze() {
	currentMaze = createMaze()
}

// Print to the terminal the average steps to solution for the current session
func printResults() {
	fmt.Printf("Labyrinth solved %d times with an avg of %d steps\n", len(scores), mazelib.AvgScores(scores))
}

// Return a room from the maze
func (m *Maze) GetRoom(x, y int) (*mazelib.Room, error) {
	if x < 0 || y < 0 || x >= m.Width() || y >= m.Height() {
		return &mazelib.Room{}, errors.New("room outside of maze boundaries")
	}

	return &m.rooms[y][x], nil
}

func (m *Maze) Width() int  { return len(m.rooms[0]) }
func (m *Maze) Height() int { return len(m.rooms) }

// Return Icarus's current position
func (m *Maze) Icarus() (x, y int) {
	return m.icarus.X, m.icarus.Y
}

// Set the location where Icarus will awake
func (m *Maze) SetStartPoint(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Treasure {
		return errors.New("can't start in the treasure")
	}

	r.Start = true
	m.icarus = mazelib.Coordinate{x, y}
	return nil
}

// Set the location of the treasure for a given maze
func (m *Maze) SetTreasure(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Start {
		return errors.New("can't have the treasure at the start")
	}

	r.Treasure = true
	m.end = mazelib.Coordinate{x, y}
	return nil
}

// Given Icarus's current location, Discover that room
// Will return ErrVictory if Icarus is at the treasure.
func (m *Maze) LookAround() (mazelib.Survey, error) {
	if m.end.X == m.icarus.X && m.end.Y == m.icarus.Y {
		fmt.Printf("Victory achieved in %d steps \n", m.StepsTaken)
		return mazelib.Survey{}, mazelib.ErrVictory
	}

	return m.Discover(m.icarus.X, m.icarus.Y)
}

// Given two points, survey the room.
// Will return error if two points are outside of the maze
func (m *Maze) Discover(x, y int) (mazelib.Survey, error) {
	if r, err := m.GetRoom(x, y); err != nil {
		return mazelib.Survey{}, nil
	} else {
		return r.Walls, nil
	}
}

// Moves Icarus's position left one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveLeft() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Left {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x-1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x - 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position right one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveRight() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Right {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x+1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x + 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position up one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveUp() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Top {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y-1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y - 1}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position down one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveDown() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Bottom {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y+1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y + 1}
	m.StepsTaken++
	return nil
}

// Creates a maze without any walls
// Good starting point for additive algorithms
func emptyMaze() *Maze {
	z := Maze{}
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	z.rooms = make([][]mazelib.Room, ySize)
	for y := 0; y < ySize; y++ {
		z.rooms[y] = make([]mazelib.Room, xSize)
		for x := 0; x < xSize; x++ {
			z.rooms[y][x] = mazelib.Room{}
		}
	}

	return &z
}

// Creates a maze with all walls
// Good starting point for subtractive algorithms
func fullMaze() *Maze {
	z := emptyMaze()
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	for y := 0; y < ySize; y++ {
		for x := 0; x < xSize; x++ {
			z.rooms[y][x].Walls = mazelib.Survey{true, true, true, true}
		}
	}

	return z
}

func (m *Maze) addBoundary() {
	for x := 0; x < m.Width(); x += 1 {
		room, _ := m.GetRoom(x, 0)
		room.AddWall(mazelib.N)
		room, _ = m.GetRoom(x, m.Height()-1)
		room.AddWall(mazelib.S)
	}
	for y := 0; y < m.Height(); y += 1 {
		room, _ := m.GetRoom(0, y)
		room.AddWall(mazelib.W)
		room, _ = m.GetRoom(m.Width()-1, y)
		room.AddWall(mazelib.E)
	}
}

func findNaiveRoute(src, dst common.Coordinate) []common.Coordinate {
	ret := []common.Coordinate{}
	for c := src; c != dst; {
		ret = append(ret, c)
		dir := rand.Intn(2)
		if c.X == dst.X {
			dir = 0
		}
		if c.Y == dst.Y {
			dir = 1
		}
		if dir == 0 {
			if c.Y < dst.Y {
				c = c.Down()
			} else {
				c = c.Up()
			}
		} else {
			if c.X < dst.X {
				c = c.Right()
			} else {
				c = c.Left()
			}
		}
	}
	return append(ret, dst)
}

type rect struct {
	X, Y, W, H int
}

func (r rect) contains(c common.Coordinate) bool {
	return r.X <= c.X && c.X < r.X+r.W && r.Y <= c.Y && c.Y < r.Y+r.H
}

func (m *Maze) toRect() rect {
	return rect{X: 0, Y: 0, W: m.Width(), H: m.Height()}
}

func (r rect) cuth(src, dst common.Coordinate) (rect, common.Coordinate, rect, common.Coordinate, bool) {
	if src.Y == dst.Y || r.H <= cutLimit {
		return rect{}, common.Coordinate{}, rect{}, common.Coordinate{}, false
	}
	cy := (src.Y+dst.Y)/2 + 1
	cx := rand.Intn(r.W) + r.X
	if cx == src.X || cx == dst.X {
		cx = rand.Intn(r.W) + r.X
	}
	r1 := rect{X: r.X, Y: r.Y, W: r.W, H: cy - r.Y}
	r2 := rect{X: r.X, Y: cy, W: r.W, H: r.H - r1.H}
	if r1.contains(src) {
		return r1, common.NewCoordinate(cx, cy-1), r2, common.NewCoordinate(cx, cy), true
	}
	return r2, common.NewCoordinate(cx, cy), r1, common.NewCoordinate(cx, cy-1), true
}

func (r rect) cutv(src, dst common.Coordinate) (rect, common.Coordinate, rect, common.Coordinate, bool) {
	if src.X == dst.X || r.W <= cutLimit {
		return rect{}, common.Coordinate{}, rect{}, common.Coordinate{}, false
	}
	cx := (src.X+dst.X)/2 + 1
	cy := rand.Intn(r.H) + r.Y
	if cy == src.Y || cy == dst.Y {
		cy = rand.Intn(r.H) + r.Y
	}
	r1 := rect{X: r.X, Y: r.Y, W: cx - r.X, H: r.H}
	r2 := rect{X: cx, Y: r.Y, W: r.W - r1.W, H: r.H}
	if r1.contains(src) {
		return r1, common.NewCoordinate(cx-1, cy), r2, common.NewCoordinate(cx, cy), true
	}
	return r2, common.NewCoordinate(cx, cy), r1, common.NewCoordinate(cx-1, cy), true
}

// cut the submaze into two pieces, so src and dst are in different piece, if possible
func (r rect) cut(src, dst common.Coordinate) (rect, common.Coordinate, rect, common.Coordinate, bool) {
	hrsrc, hcsrc, hrdst, hcdst, hok := r.cuth(src, dst)
	vrsrc, vcsrc, vrdst, vcdst, vok := r.cutv(src, dst)
	if !hok {
		return vrsrc, vcsrc, vrdst, vcdst, vok
	}
	if !vok {
		return hrsrc, hcsrc, hrdst, hcdst, hok
	}
	if r.W > r.H {
		return vrsrc, vcsrc, vrdst, vcdst, vok
	}
	return hrsrc, hcsrc, hrdst, hcdst, hok
}

// findRoute find a route in the sub-maze recursively
func (r rect) findRoute(src, dst common.Coordinate) []common.Coordinate {
	rsrc, csrc, rdst, cdst, ok := r.cut(src, dst)
	if !ok {
		return findNaiveRoute(src, dst)
	}
	return append(rsrc.findRoute(src, csrc), rdst.findRoute(cdst, dst)...)
}

func (m *Maze) contains(c common.Coordinate) bool {
	return 0 <= c.X && c.X < m.Width() && 0 <= c.Y && c.Y < m.Height()
}

func (m *Maze) addWall(c common.Coordinate, dir int) {
	nb := c.Neighbor(dir)
	if m.contains(nb) {
		r, _ := m.GetRoom(c.X, c.Y)
		r.AddWall(dir)
		r, _ = m.GetRoom(nb.X, nb.Y)
		r.AddWall(common.ReverseDirection[dir])
	}
}

func (m *Maze) sealRoom(c common.Coordinate) {
	m.addWall(c, mazelib.N)
	m.addWall(c, mazelib.S)
	m.addWall(c, mazelib.E)
	m.addWall(c, mazelib.W)
}

func (m *Maze) removeWallBetween(c1, c2 common.Coordinate) {
	d1 := c1.GetDir(c2)
	r1, _ := m.GetRoom(c1.X, c1.Y)
	r1.RmWall(d1)

	d2 := c2.GetDir(c1)
	r2, _ := m.GetRoom(c2.X, c2.Y)
	r2.RmWall(d2)
}

func (m *Maze) paveRoute(route []common.Coordinate) {
	for _, c := range route {
		m.sealRoom(c)
	}
	for i := range route {
		if i > 0 {
			m.removeWallBetween(route[i], route[i-1])
		}
	}
}

func (m *Maze) floodfill(c, from common.Coordinate, explored map[common.Coordinate]bool) {
	m.sealRoom(c)
	m.removeWallBetween(c, from)
	explored[c] = true
	for _, nb := range c.Neighbors() {
		if m.contains(nb) && !explored[nb] {
			m.floodfill(nb, c, explored)
		}
	}
}

func (m *Maze) buildMaze(src, dst common.Coordinate) {
	r := m.toRect()
	route := r.findRoute(src, dst)
	m.paveRoute(route)

	explored := make(map[common.Coordinate]bool)
	for _, c := range route {
		explored[c] = true
	}

	order := rand.Perm(len(route) - 1)
	for _, idx := range order {
		c := route[idx]
		for _, nb := range c.Neighbors() {
			if m.contains(nb) && !explored[nb] {
				m.floodfill(nb, c, explored)
			}
		}
	}
}

func createMaze() *Maze {

	m := emptyMaze()
	sx, sy := rand.Intn(m.Width()), rand.Intn(m.Height())
	dx, dy := rand.Intn(m.Width()), rand.Intn(m.Height())
	for dx == sx && dy == sy {
		dx, dy = rand.Intn(m.Width()), rand.Intn(m.Height())
	}
	m.SetStartPoint(sx, sy)
	m.SetTreasure(dx, dy)
	m.addBoundary()

	m.buildMaze(common.NewCoordinate(sx, sy), common.NewCoordinate(dx, dy))

	return m
}
