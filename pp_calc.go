package main

import "math"

const objectsToStore = 10
const NCount = 100

type PPIter struct {
	PP float64

	Skills    Skills
	Window300 float64
	Window100 float64
	Window50  float64

	ProbResult             float64
	ProbSS                 float64
	ProbN100sOr50sOrMisses [NCount]float64
	ProbN50sOrMisses       [NCount]float64
	ProbNMisses            [NCount]float64

	Expected300s   float64
	Expected100s   float64
	Expected50s    float64
	ExpectedMisses float64

	LastClicks      []Click
	ClicksDeltaTime []float64
}

// calculates lower bound of probability
func (it *PPIter) CalculateProbability(
	count100s int,
	count50s int,
	countMisses int,
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
	it.ProbResult = max(0, prob100sOr50sOrMisses+prob50sOrMisses+probMisses-2)
}

func NewPPIter(
	skills Skills,
	window300 float64,
	window100 float64,
	window50 float64,
) PPIter {
	lastClicks := make([]Click, objectsToStore)
	clicksDeltaTime := make([]float64, objectsToStore)
	for i := range objectsToStore {
		lastClicks[i] = Click{
			Pos:  CenterPos,
			Time: -1e9 + 1e6*float64(i),
		}
		clicksDeltaTime[i] = 1e6
	}

	var probN100sOr50sOrMisses [NCount]float64
	var probN50sOrMisses [NCount]float64
	var probNMisses [NCount]float64
	probN100sOr50sOrMisses[0] = 1
	probN50sOrMisses[0] = 1
	probNMisses[0] = 1

	return PPIter{
		Skills:    skills,
		Window300: window300,
		Window100: window100,
		Window50:  window50,

		ProbSS:                 1,
		ProbN100sOr50sOrMisses: probN100sOr50sOrMisses,
		ProbN50sOrMisses:       probN50sOrMisses,
		ProbNMisses:            probNMisses,

		Expected300s:   0,
		Expected100s:   0,
		Expected50s:    0,
		ExpectedMisses: 0,

		LastClicks:      lastClicks,
		ClicksDeltaTime: clicksDeltaTime,
	}
}

type Click struct {
	Pos  Vec
	Time float64
}

func IterateAction(
	it *PPIter,
	action *Action,
) {
	lastClick := it.LastClicks[len(it.LastClicks)-1]

	if action.Clickable {
		it.LastClicks = append(
			it.LastClicks,
			Click{
				Pos:  action.Pos,
				Time: action.Time,
			},
		)
		it.ClicksDeltaTime = append(
			it.ClicksDeltaTime,
			action.Time-lastClick.Time,
		)

		averageClickError := 10000 / (1 + it.Skills.Tapping.Accuracy) // sqrt(E[error ^ 2])

		probAtLeast300 := ProbErrLessThanX(averageClickError, it.Window300)
		probAtLeast100 := ProbErrLessThanX(averageClickError, it.Window100)
		probAtLeast50 := ProbErrLessThanX(averageClickError, it.Window50)

		for i := NCount - 1; i >= 1; i-- {
			it.ProbN100sOr50sOrMisses[i] = it.ProbN100sOr50sOrMisses[i]*probAtLeast300 + it.ProbN100sOr50sOrMisses[i-1]*(1-probAtLeast300)
			it.ProbN50sOrMisses[i] = it.ProbN50sOrMisses[i]*probAtLeast100 + it.ProbN50sOrMisses[i-1]*(1-probAtLeast100)
			it.ProbNMisses[i] = it.ProbNMisses[i]*probAtLeast50 + it.ProbNMisses[i-1]*(1-probAtLeast50)
		}
		it.ProbSS = it.ProbSS * probAtLeast300
		it.ProbN100sOr50sOrMisses[0] = it.ProbN100sOr50sOrMisses[0] * probAtLeast300
		it.ProbN50sOrMisses[0] = it.ProbN50sOrMisses[0] * probAtLeast100
		it.ProbNMisses[0] = it.ProbNMisses[0] * probAtLeast50

		it.Expected300s += probAtLeast300
		it.Expected100s += probAtLeast100 - probAtLeast300
		it.Expected50s += probAtLeast50 - probAtLeast100
		it.ExpectedMisses += 1 - probAtLeast50
	}
}

const b_param = 3 // must be > 2
var c_param = math.Sqrt((b_param - 1) * (b_param - 2) / 2)

func ProbErrLessThanX(avgErr float64, x float64) float64 {
	return 1 - math.Pow(1+x/(avgErr*c_param), -b_param)
}
