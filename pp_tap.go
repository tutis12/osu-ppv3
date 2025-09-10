package main

import "math"

// sqrt(E[error ^ 2]) aka unstable rate
func GetUnstableRate(
	it *PPIter,
	action *Action,
) (unstableRate float64) {
	lastClick := action.LastClicks[len(action.LastClicks)-1]

	lastClickDeltaTime := action.Time - lastClick.Time
	lastClickBPM := 15000 / lastClickDeltaTime // 50ms = 300bpm 1/4

	avgBpmTo300 := 0.0
	for i := 1; i <= fakeObjects; i++ {
		deltaTime := (action.Time - action.LastClicks[len(action.LastClicks)-i].Time)
		avgBpmTo300 = max(avgBpmTo300, float64(i)*15000/(deltaTime+it.MapConstants.Window300*2))
	}

	skillBurstBPM := math.Sqrt(it.Skills.Tapping.BurstSpeed) * 10   // 900 skill in speed = 300 bpm
	skillStreamBPM := math.Sqrt(it.Skills.Tapping.StreamSpeed) * 10 // 900 skill in speed = 300 bpm
	speedErrorFactor := 1 +
		0.1*math.Pow(lastClickBPM/skillBurstBPM, 2) +
		math.Pow(avgBpmTo300/skillStreamBPM, 3)

	lowArClickError := (1 + 0.001*it.MapConstants.Preempt/it.Skills.Reading.LowAr)

	return speedErrorFactor * lowArClickError * (10000 / (1 + 2*it.Skills.Tapping.Accuracy))
}

// unstable rate calcs
const b_param = 3 // must be > 2
var c_param = math.Sqrt((b_param - 1) * (b_param - 2) / 2)

func ProbErrLessThanX(avgErr float64, x float64) float64 {
	return 1 - math.Pow(1+x/(avgErr*c_param), -b_param)
}
