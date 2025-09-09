package main

import (
	"fmt"
	"math"
	"unsafe"
)

const skillCount = 44

type Skills struct {
	Aim     AimSkills
	Tapping TappingSkills
	Reading ReadingSkills
}

// test at runtime lol :D
func init() {
	vector := [skillCount]float64{}
	for i := range skillCount {
		vector[i] = float64(i)
	}

	skills := VectorToSkills(vector)
	vector1 := SkillsToVector(skills)
	if vector1 != vector {
		panic("aaa")
	}

	if unsafe.Sizeof(skills) != skillCount*8 {
		panic(fmt.Sprintf("%d %d", unsafe.Sizeof(skills), skillCount))
	}
}

func (skills Skills) PP() float64 {
	vec := SkillsToVector(skills)
	return PowAvg(
		vec[:],
		2,
	)
}

func PowAvg(nums []float64, pow float64) float64 {
	sum := 0.0
	for _, num := range nums {
		sum += math.Pow(num, pow)
	}
	return math.Pow(sum/float64(len(nums)), 1/pow)
}

func SkillsToVector(v Skills) [skillCount]float64 {
	return *(*[skillCount]float64)(unsafe.Pointer(&v))
}

func VectorToSkills(v [skillCount]float64) Skills {
	return *(*Skills)(unsafe.Pointer(&v))
}

type AimSkills struct {
	DistancePrecision float64    // aiming correct distance towards the object
	AnglePrecision    float64    // aiming correct angle towards the object
	CursorSpeed       [4]float64 // different cursor speeds
	Stamina           float64    // long circle chains

	SpeedIncrease float64 // increase cursor speed
	SpeedDecrease float64 // decrease cursor speed
	SpeedStop     float64 // hold still / stacks / sliders

	FlicksWideAngle  float64 // wide angle flicks
	FlicksSharpAngle float64 // sharp angle flicks
	FlicksRightAngle float64 // right angle flicks

	Cheese float64 // cheese overlaps

	AngleIncrease float64 // increase jump angle
	AngleDecrease float64 // decrease jump angle

	CW  float64 // clockwise flow
	CCW float64 // counterclockwise flow

	CWChange float64     // ability to switch circular direction
	CWHold   float64     // ability to continue circular direction
	Angles   [18]float64 // increments of 10 degrees

	Spin float64
}

type TappingSkills struct {
	Accuracy float64 // high od
	Speed    float64 // high bpm
}

type ReadingSkills struct {
	LowAr    float64
	Overlaps float64
	Density  float64
}
