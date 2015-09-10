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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/golangchallenge/gc6/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Defining the icarus command.
// This will be called as 'laybrinth icarus'
var icarusCmd = &cobra.Command{
	Use:     "icarus",
	Aliases: []string{"client"},
	Short:   "Start the laybrinth solver",
	Long: `Icarus wakes up to find himself in the middle of a Labyrinth.
  Due to the darkness of the Labyrinth he can only see his immediate cell and if
  there is a wall or not to the top, right, bottom and left. He takes one step
  and then can discover if his new cell has walls on each of the four sides.

  Icarus can connect to a Daedalus and solve many laybrinths at a time.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunIcarus()
	},
}

func init() {
	RootCmd.AddCommand(icarusCmd)
}

func RunIcarus() {
	// Run the solver as many times as the user desires.
	fmt.Println("Solving", viper.GetInt("times"), "times")
	for x := 0; x < viper.GetInt("times"); x++ {

		solveMaze()
	}

	// Once we have solved the maze the required times, tell daedalus we are done
	makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/done")
}

// Make a call to the laybrinth server (daedalus) that icarus is ready to wake up
func awake() mazelib.Survey {
	contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/awake")
	if err != nil {
		fmt.Println(err)
	}
	r := ToReply(contents)
	return r.Survey
}

// Make a call to the laybrinth server (daedalus)
// to move Icarus a given direction
// Will be used heavily by solveMaze
func Move(direction string) (mazelib.Survey, error) {
	if direction == "left" || direction == "right" || direction == "up" || direction == "down" {

		contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/move/" + direction)
		if err != nil {
			return mazelib.Survey{}, err
		}

		rep := ToReply(contents)
		if rep.Victory == true {
			fmt.Println(rep.Message)
			// os.Exit(1)
			return rep.Survey, mazelib.ErrVictory
		} else {
			return rep.Survey, errors.New(rep.Message)
		}
	}

	return mazelib.Survey{}, errors.New("invalid direction")
}

// utility function to wrap making requests to the daedalus server
func makeRequest(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// Handling a JSON response and unmarshalling it into a reply struct
func ToReply(in []byte) mazelib.Reply {
	res := &mazelib.Reply{}
	json.Unmarshal(in, &res)
	return *res
}

type Coordinate struct {
	mazelib.Coordinate
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

func (c Coordinate) Neighbor(dir string) Coordinate {
	var ret Coordinate
	switch dir {
	case "left":
		ret = c.Left()
	case "right":
		ret = c.Right()
	case "up":
		ret = c.Up()
	case "down":
		ret = c.Down()
	}
	return ret
}

type Survey struct {
	mazelib.Survey
}

func (s Survey) HasWall(dir string) bool {
	var ret bool
	switch dir {
	case "left":
		ret = s.Left
	case "right":
		ret = s.Right
	case "up":
		ret = s.Top
	case "down":
		ret = s.Bottom
	}
	return ret
}

type path struct {
	coordinates []Coordinate
	size        int
}

func newPath() *path {
	return &path{
		coordinates: []Coordinate{},
		size:        0,
	}
}

func (p *path) push(coordinate Coordinate) {
	if p.size >= len(p.coordinates) {
		p.coordinates = append(p.coordinates, coordinate)
	} else {
		p.coordinates[p.size] = coordinate
	}
	p.size += 1
}

func (p *path) top() (Coordinate, error) {
	if p.size == 0 {
		return Coordinate{}, fmt.Errorf("There's no top coordinate in the empty path object")
	}
	return p.coordinates[p.size-1], nil
}

func (p *path) backtrack(explored map[Coordinate]Survey) (Coordinate, error) {
	for i := p.size - 2; i >= 0; i -= 1 {
		c := p.coordinates[i]
		if _, _, found := pickNeighbor(c, explored); found {
			p.size = i + 1 // shrink
			return c, nil
		}
	}
	return origin(), fmt.Errorf("Couldn't find a coordinate, which is not fully explored, in the path")
}

func solveMaze() {
	// Need to start with waking up to initialize a new maze
	// You'll probably want to set this to a named value and start by figuring
	// out which step to take next

	explored := make(map[Coordinate]Survey)
	src := origin()
	explored[src] = Survey{awake()}

	path := newPath()
	path.push(src)

	for {
		icarus, _ := path.top()
		if next, direction, found := pickNeighbor(icarus, explored); found {
			survey, err := Move(direction)
			if err == mazelib.ErrVictory {
				os.Exit(1)
			}
			if err != nil {
				panic(err)
			}
			path.push(next)
			explored[next] = Survey{survey}
		} else {
			dst, err := path.backtrack(explored)
			if err != nil {
				panic(err)
			}
			goback(icarus, dst, explored)
		}
	}
}

func goback(src Coordinate, dst Coordinate, explored map[Coordinate]Survey) int {
	queue := make([]Coordinate, len(explored))
	from := make(map[Coordinate]string)
	queue[0] = dst
	from[dst] = ""
	found := false
	for i, size := 0, 1; i < size && !found; i += 1 {
		c := queue[i]
		survey := explored[c]
		for _, dir := range getDirections() {
			if survey.HasWall(dir) {
				continue
			}
			nb := src.Neighbor(dir)
			if _, nbex := explored[nb]; !nbex {
				continue
			}
			if _, searched := from[nb]; searched {
				continue
			}
			queue[size] = nb
			size += 1
			from[nb] = reverseDirection(dir)
			if nb == src {
				found = true
				break
			}
		}
	}
	if !found {
		panic("goback doesn't even find a way back")
	}
	ret := 0
	for c := src; c != dst; c.Neighbor(from[c]) {
		ret += 1
		Move(from[c])
	}
	return ret
}

func reverseDirection(direction string) string {
	var ret string
	switch direction {
	case "up":
		ret = "down"
	case "down":
		ret = "up"
	case "left":
		ret = "right"
	case "right":
		ret = "left"
	}
	return ret
}

func pickNeighbor(coordinate Coordinate, explored map[Coordinate]Survey) (Coordinate, string, bool) {
	survey := explored[coordinate]
	// todo: randomize
	for _, dir := range getDirections() {
		if !survey.HasWall(dir) {
			neighbor := coordinate.Neighbor(dir)
			if _, ok := explored[neighbor]; !ok {
				return neighbor, dir, true
			}
		}
	}
	return coordinate, "", false
}

func getDirections() []string {
	return []string{"left", "right", "up", "down"}
}

func origin() Coordinate {
	return Coordinate{mazelib.Coordinate{X: 0, Y: 0}}
}
