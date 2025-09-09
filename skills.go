package main

import "math"

const skillCount = 36

type Skills struct {
	Aim     AimSkills
	Tapping TappingSkills
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

func VectorToSkills(skillVector [skillCount]float64) Skills {
	return Skills{
		Aim: AimSkills{
			Precision:        skillVector[0],
			Speed:            skillVector[1],
			Stamina:          skillVector[2],
			SpeedIncrease:    skillVector[3],
			SpeedDecrease:    skillVector[4],
			SpeedStop:        skillVector[5],
			FlicksWideAngle:  skillVector[6],
			FlicksSharpAngle: skillVector[7],
			FlicksRightAngle: skillVector[8],
			Cheese:           skillVector[9],
			AngleIncrease:    skillVector[10],
			AngleDecrease:    skillVector[11],
			CW:               skillVector[12],
			CCW:              skillVector[13],
			CWChange:         skillVector[14],
			CWHold:           skillVector[15],
			Angles:           [18]float64(skillVector[16:34]),
		},
		Tapping: TappingSkills{
			Accuracy: skillVector[34],
			Speed:    skillVector[35],
		},
	}
}

func SkillsToVector(skills Skills) [skillCount]float64 {
	return [skillCount]float64{
		skills.Aim.Precision,
		skills.Aim.Speed,
		skills.Aim.Stamina,
		skills.Aim.SpeedIncrease,
		skills.Aim.SpeedDecrease,
		skills.Aim.SpeedStop,
		skills.Aim.FlicksWideAngle,
		skills.Aim.FlicksSharpAngle,
		skills.Aim.FlicksRightAngle,
		skills.Aim.Cheese,
		skills.Aim.AngleIncrease,
		skills.Aim.AngleDecrease,
		skills.Aim.CW,
		skills.Aim.CCW,
		skills.Aim.CWChange,
		skills.Aim.CWHold,
		skills.Aim.Angles[0],
		skills.Aim.Angles[1],
		skills.Aim.Angles[2],
		skills.Aim.Angles[3],
		skills.Aim.Angles[4],
		skills.Aim.Angles[5],
		skills.Aim.Angles[6],
		skills.Aim.Angles[7],
		skills.Aim.Angles[8],
		skills.Aim.Angles[9],
		skills.Aim.Angles[10],
		skills.Aim.Angles[11],
		skills.Aim.Angles[12],
		skills.Aim.Angles[13],
		skills.Aim.Angles[14],
		skills.Aim.Angles[15],
		skills.Aim.Angles[16],
		skills.Aim.Angles[17],
		skills.Tapping.Accuracy,
		skills.Tapping.Speed,
	}
}

type AimSkills struct {
	Precision float64 // high cs
	Speed     float64 // high bpm
	Stamina   float64 // long circle chains

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
}

type TappingSkills struct {
	Accuracy float64 // high od
	Speed    float64 // high bpm
}
