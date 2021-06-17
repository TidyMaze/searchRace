package main

import (
	"fmt"
	"image/color"
	"strconv"

	// "image/color"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/go-p5/p5"
)

const MAP_WIDTH = 16000
const MAP_HEIGHT = 9000
const SCALE = 0.1
const CP_RADIUS = 600
const CP_DIAMETER = CP_RADIUS * 2
const PADDING = 1000
const MAX_ANGLE_DIFF_DEGREE = 18
const MAPS_PANEL_SIZE = 30
const MAX_LAP = 3

const POPULATION_SIZE = 50

var fastSim = true
var displayCheckpointsMapIndex int
var displayLap int
var displayCheckpoints []Coord
var displayTarget Coord

var displayCar Car

// var idxCheckpoint = 0
var allMaps [][]Coord
var thisMapSteps = 0
var totalSteps = 0

type Car struct {
	coord Coord
	vel   Vector
	angle float64
}

type Coord struct {
	x, y float64
}

type Vector struct {
	x, y float64
}

type State struct {
	car               Car
	idxCheckpoint     int
	lap               int
	passedCheckpoints int
}

type Action struct {
	thrust             int
	angle              int
	offsetAngleDegrees int
}

type Trajectory struct {
	history      []Action
	currentState State
	score        float64
}

func isSameCar(c1 Car, c2 Car) bool {
	return int(c1.angle) == int(c2.angle) &&
		int(c1.coord.x) == int(c2.coord.x) &&
		int(c1.coord.y) == int(c2.coord.y) &&
		int(c1.vel.x) == int(c2.vel.x) &&
		int(c1.vel.y) == int(c2.vel.y)
}

func isSameState(s1 State, s2 State) bool {
	return isSameCar(s1.car, s2.car) &&
		s1.idxCheckpoint == s2.idxCheckpoint &&
		s1.lap == s2.lap &&
		s1.passedCheckpoints == s2.passedCheckpoints
}

func log(msg string, v interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %+v\n", msg, v)
}

func toRadians(a float64) float64 {
	return regularizeAngle(float64(a * math.Pi / 180))
}

func toDegrees(a float64) float64 { return float64(a * 180 / math.Pi) }

func regularizeAngle(a float64) float64 {
	if a > math.Pi {
		a -= 2 * math.Pi
	}
	if a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func regularizeAngleDegree(a float64) float64 {
	if a > 180 {
		a -= 360
	}
	if a < -180 {
		a += 360
	}
	return a
}

func diffAngle(a1 float64, a2 float64) float64 {
	return regularizeAngle(a2 - a1)
}

func restrictAngle(current float64, requested float64) float64 {
	return clampAngle(requested, current-toRadians(MAX_ANGLE_DIFF_DEGREE), current+toRadians(MAX_ANGLE_DIFF_DEGREE))
}

func clampAngle(a float64, minA float64, maxA float64) float64 {
	if diffAngle(minA, a) >= 0 && diffAngle(a, maxA) >= 0 {
		return a
	}
	if math.Abs(diffAngle(a, minA)) <= math.Abs(diffAngle(a, maxA)) {
		return minA
	} else {
		return maxA
	}
}

func initCar() Car {
	return Car{
		coord: Coord{
			x: float64(randInt(0+PADDING, MAP_WIDTH-PADDING)),
			y: float64(randInt(0+PADDING, MAP_HEIGHT-PADDING)),
		},
		vel: Vector{
			x: 0,
			y: 0,
		},
		angle: 0,
	}
}

func setup() {
	rand.Seed(time.Now().UnixNano())
	p5.Canvas(MAP_WIDTH*SCALE+500, MAP_HEIGHT*SCALE)
	p5.Background(color.Gray{Y: 220})

	allMaps = make([][]Coord, 0, MAPS_PANEL_SIZE)
	for i := 0; i < MAPS_PANEL_SIZE; i++ {
		allMaps = append(allMaps, randomMap())
	}

	go searchCarParams()
}

func randInt(min int, max int) int {
	return rand.Intn(max-min) + min
}

func dist(c1 Coord, c2 Coord) float64 {
	return math.Sqrt(math.Pow(c2.x-c1.x, 2) + math.Pow(c2.y-c1.y, 2))
}

func oneCPIsTooClose(cps []Coord, c Coord) bool {
	const MIN_SPACING = 1200
	for i := 0; i < len(cps); i++ {
		if dist(cps[i], c) < MIN_SPACING {
			return true
		}
	}
	return false
}

func randomMap() []Coord {
	cpCount := randInt(3, 9)
	res := make([]Coord, 0, cpCount)
	for iCheckpoint := 0; iCheckpoint < cpCount; iCheckpoint++ {

		randCoord := Coord{float64(-1), float64(-1)}

		for randCoord.x == -1 || randCoord.y == -1 || oneCPIsTooClose(res, randCoord) {
			randCoord.x = float64(randInt(0+PADDING, MAP_WIDTH-PADDING))
			randCoord.y = float64(randInt(0+PADDING, MAP_HEIGHT-PADDING))
		}

		res = append(res, randCoord)
	}
	return res
}

func drawCheckpoints(checkpoints []Coord) {
	p5.Fill(color.White)
	p5.TextSize(24)
	for i := 0; i < len(checkpoints); i++ {
		x := checkpoints[i].x * SCALE
		y := checkpoints[i].y * SCALE
		p5.Circle(x, y, CP_DIAMETER*SCALE)
		p5.Text(strconv.Itoa(i), x, y)
	}
}

func drawCar(car Car) {
	p5.Fill(color.RGBA{R: 255, A: 255})
	p5.Circle(car.coord.x*SCALE, car.coord.y*SCALE, 50)
}

func norm(v Vector) float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y)
}

func normalVector(v Vector) Vector {
	n := norm(v)

	if n > 0 {
		return Vector{
			x: v.x / n,
			y: v.y / n,
		}
	} else {
		return Vector{
			x: 1,
			y: 0,
		}
	}
}

func multVector(v Vector, factor float64) Vector {
	return Vector{
		x: v.x * factor,
		y: v.y * factor,
	}
}

func truncVector(v Vector) Vector {
	return Vector{
		x: math.Trunc(v.x),
		y: math.Trunc(v.y),
	}
}

func truncCoord(c Coord) Coord {
	return Coord{
		x: math.Trunc(c.x),
		y: math.Trunc(c.y),
	}
}

func addVector(v1 Vector, v2 Vector) Vector {
	return Vector{
		x: v1.x + v2.x,
		y: v1.y + v2.y,
	}
}

func applyVector(c Coord, v Vector) Coord {
	return Coord{
		x: c.x + v.x,
		y: c.y + v.y,
	}
}

func vectorBetween(c1 Coord, c2 Coord) Vector {
	return Vector{
		c2.x - c1.x,
		c2.y - c1.y,
	}
}

func assert(v float64, v2 float64) {
	diff := v2 - v
	if math.Abs(diff) > 0.0000001 {
		panic(fmt.Sprintf("%f did not equal %f diff %.100f", v, v2, diff))
	}
}

func normalVectorFromAngle(a float64) Vector {
	return Vector{
		x: math.Cos(a),
		y: math.Sin(a),
	}
}

func searchCarParams() {
	cnt := 0

	for {
		cnt += 1

		totalSteps = 0
		for checkpointsMapIndex := 0; checkpointsMapIndex < len(allMaps); checkpointsMapIndex += 1 {

			displayCheckpoints = allMaps[checkpointsMapIndex]
			displayCheckpointsMapIndex = checkpointsMapIndex

			state := State{
				car:               initCar(),
				idxCheckpoint:     0,
				lap:               0,
				passedCheckpoints: 0,
			}

			displayLap = 0

			thisMapSteps = 0

			for over, turn := false, 0; !over; turn += 1 {
				over, state = update(turn, state, checkpointsMapIndex)

				displayCar = state.car

				if !fastSim {
					waitTime := 20000 * time.Microsecond
					time.Sleep(time.Duration(waitTime))
				}
			}

			// log("Done map ", fmt.Sprintf("map %d in %d steps", checkpointsMapIndex, thisMapSteps))
		}
	}
	os.Exit(0)
}

func applyAction(car Car, angle float64, thrust int) Car {
	toTargetAngleRestricted := restrictAngle(toRadians(car.angle), angle)
	car.angle = toDegrees(toTargetAngleRestricted)
	acc := multVector(normalVectorFromAngle(toTargetAngleRestricted), float64(thrust))
	car.vel = addVector(car.vel, acc)
	car.coord = applyVector(car.coord, car.vel)
	car.vel = multVector(car.vel, 0.85)
	car.vel = truncVector(car.vel)
	car.angle = math.Round(car.angle)
	car.coord = truncCoord(car.coord)

	return car
}

func applyActionOnState(checkpoints []Coord, state State, angle float64, thrust int) State {
	newCar := applyAction(state.car, angle, thrust)

	target := checkpoints[state.idxCheckpoint]
	dTarget := dist(Coord{newCar.coord.x, newCar.coord.y}, target)

	newLap := state.lap
	newCheckpointIndex := state.idxCheckpoint
	newPassedCheckpoints := state.passedCheckpoints

	if dTarget <= CP_RADIUS {
		if state.idxCheckpoint == 0 {
			newLap += 1
		}
		newCheckpointIndex = (newCheckpointIndex + 1) % len(checkpoints)
		newPassedCheckpoints += 1
	}

	return State{
		car:               newCar,
		idxCheckpoint:     newCheckpointIndex,
		lap:               newLap,
		passedCheckpoints: newPassedCheckpoints,
	}
}

func update(turn int, state State, checkpointsMapIndex int) (bool, State) {

	checkpoints := allMaps[checkpointsMapIndex]

	turnStart := getTime()

	bestAction := beamSearch(turn, turnStart, checkpoints, state)

	displayTarget = state.car.coord

	log("output", fmt.Sprintf("Turn %d best action is %+v with target %d", turn, bestAction, state.idxCheckpoint))

	newState := applyActionOnState(checkpoints, state, toRadians(float64(bestAction.angle)), bestAction.thrust)

	newState.passedCheckpoints = 0

	return newState.lap == 3, newState
}

func drawStats(checkpointsMapIndex int, lap int) {
	p5.Text(fmt.Sprintf("totalStep %d\nstep %d\nmap %d/%d\nlap %d/%d", totalSteps, thisMapSteps, checkpointsMapIndex+1, MAPS_PANEL_SIZE, lap, MAX_LAP), 10, 50)
}

func drawTarget(from Coord, to Coord) {
	p5.Line(from.x*SCALE, from.y*SCALE, to.x*SCALE, to.y*SCALE)
}

func draw() {
	if len(displayCheckpoints) > 0 {
		drawCheckpoints(displayCheckpoints)
	}
	drawCar(displayCar)

	drawTarget(Coord{displayCar.coord.x, displayCar.coord.y}, displayTarget)

	drawStats(displayCheckpointsMapIndex, displayLap)
}

func seenState(ts []Trajectory, s State) bool {
	for i := 0; i < len(ts); i++ {
		if isSameState(ts[i].currentState, s) {
			return true
		}
	}
	return false
}

func getTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func getElapsedMs(start int64) int64 {
	return (getTime() - start)
}

func timeout(curTurn int, start int64) bool {
	elapsed := getElapsedMs(start)
	maxAllowed := 0
	if curTurn == 0 {
		maxAllowed = 50
	} else {
		maxAllowed = 50
	}

	if elapsed >= int64(maxAllowed) {
		log("exit at", elapsed)
		return true
	} else {
		return false
	}
}

func beamSearch(turn int, turnStart int64, checkpoints []Coord, state State) Action {

	population := make([]Trajectory, 0, POPULATION_SIZE)

	for iCandidate := 0; iCandidate < POPULATION_SIZE; iCandidate++ {
		population = append(population, Trajectory{
			history:      []Action{},
			currentState: state,
			score:        -1,
		})
	}

	newCandidates := make([]Trajectory, 0, POPULATION_SIZE*1000)

	exitTimeout := false

	depth := 0
	for depth = 0; !exitTimeout; depth += 1 {
		// log("Depth", fmt.Sprintf("%d: %d candidates", depth, len(population)))

		seen := 0

		newCandidates = newCandidates[:0]
		for iCandidate := 0; iCandidate < len(population) && !exitTimeout; iCandidate += 1 {
			candidate := population[iCandidate]
			for offsetAngle := -18; offsetAngle <= 18; offsetAngle += 36 {
				angle := regularizeAngle(toRadians(float64(offsetAngle)) + toRadians(candidate.currentState.car.angle))
				for thrust := 0; thrust <= 200; thrust += 200 {
					newState := applyActionOnState(checkpoints, candidate.currentState, angle, thrust)

					newHistory := make([]Action, len(candidate.history), len(candidate.history)+1)
					copy(newHistory, candidate.history)
					newHistory = append(newHistory, Action{
						thrust:             thrust,
						angle:              int(toDegrees(angle)),
						offsetAngleDegrees: offsetAngle,
					})

					if !seenState(newCandidates, newState) {
						newCandidates = append(newCandidates, Trajectory{
							history:      newHistory,
							currentState: newState,
							score:        float64(newState.passedCheckpoints)*100000 - dist(newState.car.coord, checkpoints[newState.idxCheckpoint]),
						})
					} else {
						// log("already seen", fmt.Sprintf("#%d: %v", seen, newState))
						seen += 1
					}
				}
			}

			if timeout(turn, turnStart) {
				exitTimeout = true
			}
		}

		if !timeout(turn, turnStart) {

			// log("population before sort", len(newCandidates))
			// log("skipped", seen)

			sort.Slice(newCandidates, func(i, j int) bool {
				return newCandidates[i].score > newCandidates[j].score
			})
		}

		if !timeout(turn, turnStart) {

			// if depth == 7 {
			// 	for i := 0; i < 10; i++ {
			// 		log("candidate", fmt.Sprintf("%d %f %v %v", i, newCandidates[i].score, newCandidates[i].history, newCandidates[i].currentState))
			// 	}
			// }

			copy(population, newCandidates)
		}
	}

	// log("population sorted", fmt.Sprintf("pop %+v", population))

	best := population[0]

	log("best", fmt.Sprintf("cp %d at depth %d: %v ", best.currentState.passedCheckpoints, depth, best))

	return best.history[0]
}

func mainCG() {
	// checkpoints: Count of checkpoints to read
	var checkpoints int
	fmt.Scan(&checkpoints)

	checkpointsList := make([]Coord, 0, checkpoints)

	for i := 0; i < checkpoints; i++ {
		// checkpointX: Position X
		// checkpointY: Position Y
		var checkpointX, checkpointY int
		fmt.Scan(&checkpointX, &checkpointY)
		checkpointsList = append(checkpointsList, Coord{float64(checkpointX), float64(checkpointY)})
	}

	turn := 0
	turnStart := int64(0)

	for ; ; turn += 1 {
		// checkpointIndex: Index of the checkpoint to lookup in the checkpoints input, initially 0
		// x: Position X
		// y: Position Y
		// vx: horizontal speed. Positive is right
		// vy: vertical speed. Positive is downwards
		// angle: facing angle of this car
		var checkpointIndex, x, y, vx, vy, angle int
		fmt.Scan(&checkpointIndex, &x, &y, &vx, &vy, &angle)

		turnStart = getTime()

		currentCar := Car{
			vel: Vector{
				x: float64(vx),
				y: float64(vy),
			},
			coord: Coord{
				x: float64(x),
				y: float64(y),
			},
			angle: float64(angle),
		}

		state := State{
			car:               currentCar,
			idxCheckpoint:     checkpointIndex,
			lap:               0,
			passedCheckpoints: 0,
		}

		bestAction := beamSearch(turn, turnStart, checkpointsList, state)

		log("output", fmt.Sprintf("best action is %+v with target %d", bestAction, state.idxCheckpoint))

		// fmt.Fprintln(os.Stderr, "Debug messages...")
		offsetAngle := bestAction.offsetAngleDegrees
		log("offsetAngle", offsetAngle)
		fmt.Printf("EXPERT %d %d\n", offsetAngle, int(bestAction.thrust))
	}
}

func main() {

	assert(toRadians(0), 0)
	assert(toRadians(180), math.Pi)
	assert(toRadians(360), 0)
	assert(toRadians(390), toRadians(30))
	assert(toRadians(-180), -math.Pi)
	assert(toRadians(-360), 0)

	assert(diffAngle(toRadians(10), toRadians(30)), toRadians(20))
	assert(diffAngle(toRadians(30), toRadians(10)), toRadians(-20))
	assert(diffAngle(toRadians(10), toRadians(-30)), toRadians(-40))
	assert(diffAngle(toRadians(10), toRadians(330)), toRadians(-40))
	assert(diffAngle(toRadians(350), toRadians(-350)), toRadians(20))

	assert(clampAngle(toRadians(30), toRadians(20), toRadians(40)), toRadians(30))
	assert(clampAngle(toRadians(20), toRadians(20), toRadians(40)), toRadians(20))
	assert(clampAngle(toRadians(10), toRadians(20), toRadians(40)), toRadians(20))
	assert(clampAngle(toRadians(50), toRadians(20), toRadians(40)), toRadians(40))

	assert(restrictAngle(toRadians(30), toRadians(30)), toRadians(30))
	assert(restrictAngle(toRadians(30), toRadians(0)), toRadians(12))
	assert(restrictAngle(toRadians(30), toRadians(60)), toRadians(48))

	p5.Run(setup, draw)
	// mainCG()
}
