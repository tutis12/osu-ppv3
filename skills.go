package main

import (
	"fmt"
	"math"
	"unsafe"
)

const skillCount = 8

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
	DistancePrecision float64 // aiming correct distance towards the object
	AnglePrecision    float64 // aiming correct angle towards the object
	Spin              float64 // spinners
}

type TappingSkills struct {
	Accuracy    float64 // high od
	BurstSpeed  float64 // high bpm
	StreamSpeed float64 // high bpm
	HoldSliders float64 // ability to hold sliderss
}

type ReadingSkills struct {
	LowAr float64
}
