package main

import (
	"math"
)

type sample struct {
	SkillVector [skillCount]float64
	PPIter      PPIter
}

func scaleSkills(skills [skillCount]float64, factor float64) [skillCount]float64 {
	for i := range skillCount {
		skills[i] = min(1e30, max(1, skills[i]*factor))
	}
	return skills
}

func GradientDescent(
	fn func(Skills) PPIter,
) PPIter {
	const targetProbability = 0.1 // try to make probability of result 10%

	scaleSample := func(x sample) sample {
		var underExists, overExists bool
		var underSample, overSample sample
		if x.PPIter.ProbResult < targetProbability {
			underExists = true
			underSample = x
		} else {
			overExists = true
			overSample = x
		}
		for range 1000 {
			var nextVector [skillCount]float64
			if !underExists {
				nextVector = scaleSkills(overSample.SkillVector, 0.01)
			} else if !overExists {
				nextVector = scaleSkills(underSample.SkillVector, 100)
			} else {
				for i := range skillCount {
					nextVector[i] = (underSample.SkillVector[i] + overSample.SkillVector[i]) / 2
				}
			}
			nextSample := sample{
				SkillVector: nextVector,
				PPIter:      fn(VectorToSkills(nextVector)),
			}
			if nextSample.PPIter.ProbResult < targetProbability {
				underExists = true
				underSample = nextSample
			} else {
				overExists = true
				overSample = nextSample
			}
			if math.Abs(nextSample.PPIter.ProbResult-targetProbability) < 1e-5 {
				break
			}
		}
		if overExists && underExists {
			if math.Abs(overSample.PPIter.ProbResult-targetProbability) <
				math.Abs(underSample.PPIter.ProbResult-targetProbability) {
				return overSample
			} else {
				return underSample
			}
		} else if overExists {
			return overSample
		} else {
			return underSample
		}
	}

	var ret sample
	{
		for i := range skillCount {
			ret.SkillVector[i] = 1e9
		}
		ret.PPIter = fn(VectorToSkills(ret.SkillVector))
		ret = scaleSample(ret)
	}

	maxDelta := 1.0
	for _, skill := range ret.SkillVector {
		maxDelta = max(maxDelta, skill*2)
	}

	for delta := maxDelta; delta >= 0.01; delta /= 2 {
		for _, sign := range []float64{-1, 1} {
			improved := true
			for improved {
				improved = false
				lastPP := ret.PPIter.PP
				for i := range skillCount {
					newSample := sample{
						SkillVector: ret.SkillVector,
					}
					newSample.SkillVector[i] = max(1, newSample.SkillVector[i]+delta*sign)

					newSample.PPIter = fn(VectorToSkills(newSample.SkillVector))
					newSample = scaleSample(newSample)

					if newSample.PPIter.PP < ret.PPIter.PP {
						if newSample.PPIter.PP < lastPP-1e-3 {
							improved = true
						}
						ret = newSample
					}
				}
			}
		}
	}

	return ret.PPIter
}
