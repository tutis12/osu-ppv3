package main

import "math"

func ProbabilityToAim(
	it *PPIter,
	action *Action,
	unstableRate float64,
) float64 {
	lastAim := action.LastAims[len(action.LastAims)-1]
	distance := max(1, Distance(lastAim.Pos, action.Pos))

	deltaTime := max(1, action.Time-lastAim.Time)

	jumpBpm := 30000 / deltaTime // 100ms = 300bpm 1/2

	radius := action.Radius

	expectedDistanceError := 0.001 * distance * jumpBpm / math.Pow(it.Skills.Aim.DistancePrecision, 0.5)

	expectedAngleError := 30 / (1 + it.Skills.Aim.AnglePrecision)

	if action.Clickable {
		timeOverObject := deltaTime * radius / distance // time over object assuming constant cursor speed
		expectedDistanceError *= 1 + 0.1*unstableRate/timeOverObject
		expectedAngleError *= 1 + 0.001*unstableRate/timeOverObject
	}

	return ProbErrLessThanX(expectedDistanceError, radius) *
		ProbErrLessThanX(expectedAngleError, radius/distance)
}

func ProbabilitiesToAimAndTap(
	it *PPIter,
	action *Action,
) (atLeast300, atLeast100, atLeast50 float64) {
	unstableRate := GetUnstableRate(
		it,
		action,
	)
	pAim := ProbabilityToAim(
		it,
		action,
		unstableRate,
	)
	atLeast300 = pAim * ProbErrLessThanX(unstableRate, it.Window300)
	atLeast100 = pAim * ProbErrLessThanX(unstableRate, it.Window100)
	atLeast50 = pAim * ProbErrLessThanX(unstableRate, it.Window50)
	return
}
