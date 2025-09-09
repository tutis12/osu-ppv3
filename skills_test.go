package main

import "testing"

func TestSkillsVector(t *testing.T) {
	vector := [skillCount]float64{}
	for i := range skillCount {
		vector[i] = float64(i)
	}

	skills := VectorToSkills(vector)
	vector1 := SkillsToVector(skills)
	if vector1 != vector {
		panic("aaa")
	}
}
