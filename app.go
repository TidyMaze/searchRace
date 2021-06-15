package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
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

var minFastThrust = 10
var maxFastThrust = 200
var minSlowThrust = 10
var maxSlowThrust = 200
var minMaxAngle = 0
var maxMaxAngle = 180

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
	car           Car
	idxCheckpoint int
	lap           int
}

type Action struct {
	thrust int
	angle  int
}

type Trajectory struct {
	history      []Action
	currentState State
	score        float64
}

func log(msg string, v interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, v)
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
				car:           initCar(),
				idxCheckpoint: 0,
				lap:           0,
			}

			displayLap = 0

			thisMapSteps = 0

			for over := false; !over; {
				over, state = update(state, checkpointsMapIndex)

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
	car.vel = addVector(Vector{car.vel.x, car.vel.y}, acc)
	car.coord = applyVector(car.coord, car.vel)
	car.vel = multVector(car.vel, 0.85)
	car.vel = truncVector(car.vel)

	car.coord.x = math.Trunc(car.coord.x)
	car.coord.y = math.Trunc(car.coord.y)

	return car
}

func update(state State, checkpointsMapIndex int) (bool, State) {

	target := allMaps[checkpointsMapIndex][state.idxCheckpoint]

	bestAction := beamSearch(allMaps[checkpointsMapIndex], state)

	displayTarget = applyVector(state.car.coord, normalVectorFromAngle(toRadians(float64(bestAction.angle))))

	// log("output", fmt.Sprintf("Going to %+v at thrust %d", targetCoord, outputThrust))

	state.car = applyAction(state.car, toRadians(float64(bestAction.angle)), bestAction.thrust)

	thisMapSteps += 1
	totalSteps += 1

	dTarget := dist(Coord{state.car.coord.x, state.car.coord.y}, target)

	if (dTarget <= CP_RADIUS && state.lap == MAX_LAP && state.idxCheckpoint == 0) || thisMapSteps > 600 {
		return true, state
	} else if dTarget <= CP_RADIUS {
		if state.idxCheckpoint == 0 {
			state.lap += 1
			displayLap = state.lap
		}
		state.idxCheckpoint = (state.idxCheckpoint + 1) % len(allMaps[checkpointsMapIndex])
		return false, state
	} else {
		return false, state
	}
}

func drawStats(checkpointsMapIndex int, lap int) {
	p5.Text(fmt.Sprintf("totalStep %d\nstep %d\nmap %d/%d\nlap %d/%d", totalSteps, thisMapSteps, checkpointsMapIndex+1, MAPS_PANEL_SIZE, lap, MAX_LAP), 10, 50)
}

func drawSearchSpace() {
	p5.Fill(color.Transparent)
	p5.Rect(float64(1600+minFastThrust), float64(100+minSlowThrust), float64(maxFastThrust)-float64(minFastThrust), float64(maxSlowThrust)-float64(minSlowThrust))
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
	drawSearchSpace()
}

func beamSearch(checkpoints []Coord, state State) Action {

	targetCheckpoint := checkpoints[checkpointIndex]

	bestAction := Action{}

	return bestAction
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
	for {
		// checkpointIndex: Index of the checkpoint to lookup in the checkpoints input, initially 0
		// x: Position X
		// y: Position Y
		// vx: horizontal speed. Positive is right
		// vy: vertical speed. Positive is downwards
		// angle: facing angle of this car
		var checkpointIndex, x, y, vx, vy, angle int
		fmt.Scan(&checkpointIndex, &x, &y, &vx, &vy, &angle)

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

		thrust, targetCheckpoint := beamSearch(checkpointsList, checkpointIndex, currentCar)

		// fmt.Fprintln(os.Stderr, "Debug messages...")
		fmt.Printf("%d %d %d message\n", int(targetCheckpoint.x), int(targetCheckpoint.y), thrust)
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
}
