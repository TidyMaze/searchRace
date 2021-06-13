package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-p5/p5"
)

const MAP_WIDTH = 16000
const MAP_HEIGHT = 9000
const SCALE = 0.1
const CP_DIAMETER = 600 * 2
const PADDING = 1000
const MAX_ANGLE_DIFF_DEGREE = 18
const MAPS_PANEL_SIZE = 100

var checkpointsMapIndex int
var car Car
var idxCheckpoint = 0
var allMaps [][]Coord
var thisMapSteps = 0
var totalSteps = 0

var latestMapIndex int
var latestCar Car
var latestMutex sync.Mutex

var lap = 0

type Car struct {
	x, y   float64
	vx, vy float64
	angle  float64
}

type Coord struct {
	x, y float64
}

type Vector struct {
	x, y float64
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

func setup() {
	rand.Seed(time.Now().UnixNano())
	p5.Canvas(MAP_WIDTH*SCALE, MAP_HEIGHT*SCALE)
	p5.Background(color.Gray{Y: 220})

	allMaps = make([][]Coord, 0, MAPS_PANEL_SIZE)
	for i := 0; i < MAPS_PANEL_SIZE; i++ {
		allMaps = append(allMaps, randomMap())
	}

	car = Car{
		x:     float64(randInt(0+PADDING, MAP_WIDTH-PADDING)),
		y:     float64(randInt(0+PADDING, MAP_HEIGHT-PADDING)),
		vx:    0,
		vy:    0,
		angle: 0,
	}

	go udpdateLoop()
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
	cpCount := randInt(5, 10)
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
	p5.Circle(car.x*SCALE, car.y*SCALE, 50)
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

func addVector(v1 Vector, v2 Vector) Vector {
	return Vector{
		x: v1.x + v2.x,
		y: v1.y + v2.y,
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

func udpdateLoop() {
	for checkpointsMapIndex < len(allMaps) {
		update()
		totalSteps += 1
	}
	log("took", totalSteps)
}

func update() {
	// newMap = randomMap()
	thrust := 100

	target := allMaps[checkpointsMapIndex][idxCheckpoint]

	toTargetVector := Vector{
		x: target.x - car.x,
		y: target.y - car.y,
	}

	toTargetAngle := math.Atan2(toTargetVector.y, toTargetVector.x)

	toTargetAngleRestricted := restrictAngle(toRadians(car.angle), toTargetAngle)

	car.angle = toDegrees(toTargetAngleRestricted)

	acc := multVector(normalVectorFromAngle(toTargetAngleRestricted), float64(thrust))
	newSpeed := addVector(Vector{car.vx, car.vy}, acc)

	car.vx = newSpeed.x
	car.vy = newSpeed.y

	car.x += car.vx
	car.y += car.vy

	car.vx *= 0.85
	car.vy *= 0.85

	car.vx = math.Trunc(car.vx)
	car.vy = math.Trunc(car.vy)

	car.x = math.Trunc(car.x)
	car.y = math.Trunc(car.y)

	thisMapSteps += 1

	if (dist(Coord{car.x, car.y}, target) < 600 && lap == 5 && idxCheckpoint == 0) || thisMapSteps > 10000 {
		lap = 0
		idxCheckpoint = 0
		log("Done map ", fmt.Sprintf("map %d in %d steps", checkpointsMapIndex, thisMapSteps))
		checkpointsMapIndex = checkpointsMapIndex + 1
		thisMapSteps = 0
	} else if dist(Coord{car.x, car.y}, target) < 600 {
		if idxCheckpoint == 0 {
			lap += 1
		}
		idxCheckpoint = (idxCheckpoint + 1) % len(allMaps[checkpointsMapIndex])
	}
}

func draw() {
	if checkpointsMapIndex < len(allMaps) {
		drawCheckpoints(allMaps[checkpointsMapIndex])
	}
	drawCar(car)
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

		targetCheckpoint := checkpointsList[checkpointIndex]

		log("target", targetCheckpoint)

		angleCarTarget := math.Atan2(targetCheckpoint.y-float64(y), targetCheckpoint.x-float64(x))

		angleCarVelocity := math.Atan2(float64(vy), float64(vx))

		diffAngleCarTarget := diffAngle(angleCarVelocity, angleCarTarget)

		log("angleCarTarget", toDegrees(angleCarTarget))
		log("angleCarVelocity", toDegrees(angleCarVelocity))
		log("diffAngleCarTarget", toDegrees(diffAngleCarTarget))

		thrust := 200
		if toDegrees(math.Abs(diffAngleCarTarget)) > 10 {
			thrust = 80
		}

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