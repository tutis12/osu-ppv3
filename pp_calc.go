package main

type PPIter struct {
	MapConstants MapConstants
	PP           float64

	Skills Skills

	ProbResult float64

	ProbN100sOr50sOrMisses KMisses
	ProbN50sOrMisses       KMisses
	ProbNMisses            KMisses

	// lazer judgements
	ProbNSliderTickMisses KMisses
	ProbNSliderEndMisses  KMisses
	ProbNSpinnerMisses    KMisses

	SliderProbs StableSliderProbs
}

type StableSliderProbs struct {
	P300 float64
	P100 float64
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
	prob100sOr50sOrMisses := it.ProbN100sOr50sOrMisses.GetSum(count100s + count50s + countMisses)
	prob50sOrMisses := it.ProbN50sOrMisses.GetSum(count50s + countMisses)
	probMisses := it.ProbNMisses.GetSum(countMisses)

	standardJudgementsProb := max(
		0,
		prob100sOr50sOrMisses+prob50sOrMisses+probMisses-2,
	)

	if !it.MapConstants.Mods.Lazer {
		it.ProbResult = standardJudgementsProb
		return
	}

	probSliderEndMisses := it.ProbNSliderEndMisses.GetSum(countSliderEndMisses)
	probSliderTickMisses := it.ProbNSliderTickMisses.GetSum(countSliderTickMisses)
	probSpinnerMisses := it.ProbNSpinnerMisses.GetSum(countSpinnerMisses)

	it.ProbResult = probSliderEndMisses * probSliderTickMisses * probSpinnerMisses *
		standardJudgementsProb
}

func NewPPIter(
	mapConstants MapConstants,
	skills Skills,
) PPIter {
	return PPIter{
		MapConstants:           mapConstants,
		Skills:                 skills,
		ProbN100sOr50sOrMisses: NewKMisses(),
		ProbN50sOrMisses:       NewKMisses(),
		ProbNMisses:            NewKMisses(),
		ProbNSliderTickMisses:  NewKMisses(),
		ProbNSliderEndMisses:   NewKMisses(),
		ProbNSpinnerMisses:     NewKMisses(),
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
	if action.Clickable {
		atLeast300, atLeast100, atLeast50 := ProbabilitiesToAimAndTap(
			it,
			action,
		)

		if it.MapConstants.Mods.Lazer || action.Circle {
			it.ProbN100sOr50sOrMisses.Add(atLeast300)
			it.ProbN50sOrMisses.Add(atLeast100)
			it.ProbNMisses.Add(atLeast50)
		} else {
			it.SliderProbs = StableSliderProbs{
				P300: atLeast50,
				P100: 0,
			}
		}
	} else {
		pAim := ProbabilityToAim(
			it,
			action,
			1,
		)
		actionProb := pAim
		if action.Spinner {
			actionProb /= (1 + it.MapConstants.Window50/it.Skills.Aim.Spin)
		} else {
			actionProb /= (1 + 0.1/it.Skills.Tapping.HoldSliders)
		}
		if it.MapConstants.Mods.Lazer {
			if action.SliderTick {
				it.ProbNSliderTickMisses.Add(actionProb)
			} else if action.SliderEnd {
				it.ProbNSliderEndMisses.Add(actionProb)
			} else if action.Spinner {
				it.ProbNSpinnerMisses.Add(actionProb)
			} else {
				panic("unexpected case")
			}
		} else {
			if action.Spinner { // either 300 or miss for now
				it.ProbN100sOr50sOrMisses.Add(actionProb)
				it.ProbN50sOrMisses.Add(actionProb)
				it.ProbNMisses.Add(actionProb)
			} else { //part of slider
				prob := it.SliderProbs
				it.SliderProbs = StableSliderProbs{
					P300: prob.P300 * actionProb,
					P100: prob.P300*(1-actionProb) +
						prob.P100 +
						(1-prob.P300-prob.P100)*(actionProb),
				}

				if action.SliderEnd {
					prob := it.SliderProbs
					it.ProbN100sOr50sOrMisses.Add(prob.P300)
					it.ProbN50sOrMisses.Add(prob.P300 + prob.P100)
					it.ProbNMisses.Add(prob.P300 + prob.P100)
				}
			}
		}
	}
}
