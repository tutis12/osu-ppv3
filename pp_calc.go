package main

import "math"

const NCount = 100

type PPIter struct {
	PP float64

	Skills       Skills
	ApproachRate float64
	Window300    float64
	Window100    float64
	Window50     float64
	SkillBPM     float64

	ProbResult             float64
	ProbSS                 float64
	ProbN100sOr50sOrMisses [NCount]float64
	ProbN50sOrMisses       [NCount]float64
	ProbNMisses            [NCount]float64
	ProbNSliderTickMisses  [NCount]float64
	ProbNSliderEndMisses   [NCount]float64
	ProbNSpinnerMisses     [NCount]float64

	Expected300s             float64
	Expected100s             float64
	Expected50s              float64
	ExpectedMisses           float64
	ExpectedSliderTickMisses float64
	ExpectedSliderEndMisses  float64
	ExpectedSpinnerMisses    float64
}

// calculates lower bound of probability
func (it *PPIter) CalculateProbability(
	count100s int,
	count50s int,
	countMisses int,
	countSliderEndMisses int,
	countSliderTickMisses int,
	countSpinnerMisses int,
) {
	prob100sOr50sOrMisses := 0.0
	prob50sOrMisses := 0.0
	probMisses := 0.0

	for i := range count100s + count50s + countMisses + 1 {
		prob100sOr50sOrMisses += it.ProbN100sOr50sOrMisses[min(NCount-1, i)]
	}
	for i := range count50s + countMisses + 1 {
		prob50sOrMisses += it.ProbN50sOrMisses[min(NCount-1, i)]
	}
	for i := range countMisses + 1 {
		probMisses += it.ProbNMisses[min(NCount-1, i)]
	}

	probSliderEndMisses := 0.0
	probSliderTickMisses := 0.0
	probSpinnerMisses := 0.0

	for i := range countSliderEndMisses + 1 {
		probSliderEndMisses += it.ProbNSliderEndMisses[min(NCount-1, i)]
	}
	for i := range countSliderTickMisses + 1 {
		probSliderTickMisses += it.ProbNSliderTickMisses[min(NCount-1, i)]
	}
	for i := range countSpinnerMisses + 1 {
		probSpinnerMisses += it.ProbNSpinnerMisses[min(NCount-1, i)]
	}

	it.ProbResult = probSliderEndMisses * probSliderTickMisses * probSpinnerMisses *
		max(
			0,
			prob100sOr50sOrMisses+prob50sOrMisses+probMisses-2,
		)
}

func NewPPIter(
	skills Skills,
	approachRate float64,
	window300 float64,
	window100 float64,
	window50 float64,
) PPIter {

	var probN100sOr50sOrMisses [NCount]float64
	var probN50sOrMisses [NCount]float64
	var probNMisses [NCount]float64
	var probNSliderTickMisses [NCount]float64
	var probNSliderEndMisses [NCount]float64
	var probNSpinnerMisses [NCount]float64
	probN100sOr50sOrMisses[0] = 1
	probN50sOrMisses[0] = 1
	probNMisses[0] = 1
	probNSliderTickMisses[0] = 1
	probNSliderEndMisses[0] = 1
	probNSpinnerMisses[0] = 1

	return PPIter{
		Skills:       skills,
		ApproachRate: approachRate,
		Window300:    window300,
		Window100:    window100,
		Window50:     window50,
		SkillBPM:     math.Sqrt(skills.Tapping.Speed) * 10, // 900 skill in speed = 300 bpm

		ProbSS:                 1,
		ProbN100sOr50sOrMisses: probN100sOr50sOrMisses,
		ProbN50sOrMisses:       probN50sOrMisses,
		ProbNMisses:            probNMisses,
		ProbNSliderTickMisses:  probNSliderTickMisses,
		ProbNSliderEndMisses:   probNSliderEndMisses,
		ProbNSpinnerMisses:     probNSpinnerMisses,

		Expected300s:             0,
		Expected100s:             0,
		Expected50s:              0,
		ExpectedMisses:           0,
		ExpectedSliderTickMisses: 0,
		ExpectedSliderEndMisses:  0,
	}
}

type TimePos struct {
	Pos    Vec
	Radius float64
	Time   float64
}

func IterateAction(
	it *PPIter,
	action *Action,
) {
	lastClick := action.LastClicks[len(action.LastClicks)-1]

	lastAim := action.LastAims[len(action.LastAims)-1]
	distance := max(1, Distance(lastAim.Pos, action.Pos))

	radius := action.Radius

	deltaTime := action.Time - lastAim.Time
	jumpBPM := 30000 / deltaTime //100ms = 300bpm 1/2

	const baseJumpBPM = 180
	cursorSpeed0 := distance * math.Pow(jumpBPM/baseJumpBPM, 0.5)
	cursorSpeed1 := distance * math.Pow(jumpBPM/baseJumpBPM, 1)
	cursorSpeed2 := distance * math.Pow(jumpBPM/baseJumpBPM, 1.5)
	cursorSpeed3 := distance * math.Pow(jumpBPM/baseJumpBPM, 2)

	expectedDistanceError := 20 / (PowAvg(
		[]float64{it.Skills.Aim.CursorSpeed[0] / cursorSpeed0,
			it.Skills.Aim.CursorSpeed[1] / cursorSpeed1,
			it.Skills.Aim.CursorSpeed[2] / cursorSpeed2,
			it.Skills.Aim.CursorSpeed[3] / cursorSpeed3,
		},
		2,
	) * (1 + it.Skills.Aim.DistancePrecision/10000))

	expectedAngleError := 30 / (1 + it.Skills.Aim.AnglePrecision)

	probabilityToAim := ProbErrLessThanX(expectedDistanceError, radius) *
		ProbErrLessThanX(expectedAngleError, radius/distance)

	if action.Clickable {
		lastClickDeltaTime := action.Time - lastClick.Time
		lastClickBPM := 15000 / lastClickDeltaTime // 50ms = 300bpm 1/4

		speedErrorFactor := (1 + 0.1*math.Pow(lastClickBPM/it.SkillBPM, 3))

		lowArClickError := (1 + 0.1*max(0, 11-it.ApproachRate)/it.Skills.Reading.LowAr)

		// sqrt(E[error ^ 2]) aka unstable rate
		averageClickError := speedErrorFactor * lowArClickError * (10000 / (1 + 2*it.Skills.Tapping.Accuracy))

		probAtLeast300 := probabilityToAim * ProbErrLessThanX(averageClickError, it.Window300)
		probAtLeast100 := probabilityToAim * ProbErrLessThanX(averageClickError, it.Window100)
		probAtLeast50 := probabilityToAim * ProbErrLessThanX(averageClickError, it.Window50)

		for i := NCount - 1; i >= 1; i-- {
			it.ProbN100sOr50sOrMisses[i] = it.ProbN100sOr50sOrMisses[i]*probAtLeast300 +
				it.ProbN100sOr50sOrMisses[i-1]*(1-probAtLeast300)
			it.ProbN50sOrMisses[i] = it.ProbN50sOrMisses[i]*probAtLeast100 +
				it.ProbN50sOrMisses[i-1]*(1-probAtLeast100)
			it.ProbNMisses[i] = it.ProbNMisses[i]*probAtLeast50 +
				it.ProbNMisses[i-1]*(1-probAtLeast50)
		}
		it.ProbSS = it.ProbSS * probAtLeast300
		it.ProbN100sOr50sOrMisses[0] = it.ProbN100sOr50sOrMisses[0] * probAtLeast300
		it.ProbN50sOrMisses[0] = it.ProbN50sOrMisses[0] * probAtLeast100
		it.ProbNMisses[0] = it.ProbNMisses[0] * probAtLeast50

		it.Expected300s += probAtLeast300
		it.Expected100s += probAtLeast100 - probAtLeast300
		it.Expected50s += probAtLeast50 - probAtLeast100
		it.ExpectedMisses += 1 - probAtLeast50
	} else if action.SliderTick {
		for i := NCount - 1; i >= 1; i-- {
			it.ProbNSliderTickMisses[i] = it.ProbNSliderTickMisses[i]*probabilityToAim +
				it.ProbN100sOr50sOrMisses[i-1]*(1-probabilityToAim)
		}
		it.ProbNSliderTickMisses[0] = it.ProbNSliderTickMisses[0] * probabilityToAim

		it.ExpectedSliderTickMisses += (1 - probabilityToAim)
	} else if action.SliderEnd {
		for i := NCount - 1; i >= 1; i-- {
			it.ProbNSliderEndMisses[i] = it.ProbNSliderEndMisses[i]*probabilityToAim +
				it.ProbNSliderEndMisses[i-1]*(1-probabilityToAim)
		}
		it.ProbNSliderEndMisses[0] = it.ProbNSliderEndMisses[0] * probabilityToAim

		it.ExpectedSliderEndMisses += (1 - probabilityToAim)
	} else if action.Spinner {
		probabilityToAim /= (1 + it.Window50/it.Skills.Aim.Spin)
		for i := NCount - 1; i >= 1; i-- {
			it.ProbNSpinnerMisses[i] = it.ProbNSpinnerMisses[i]*probabilityToAim +
				it.ProbNSpinnerMisses[i-1]*(1-probabilityToAim)
		}
		it.ProbNSpinnerMisses[0] = it.ProbNSpinnerMisses[0] * probabilityToAim

		it.ExpectedSpinnerMisses += (1 - probabilityToAim)
	} else {
		panic("unexpected case")
	}
}

const b_param = 3 // must be > 2
var c_param = math.Sqrt((b_param - 1) * (b_param - 2) / 2)

func ProbErrLessThanX(avgErr float64, x float64) float64 {
	return 1 - math.Pow(1+x/(avgErr*c_param), -b_param)
}
