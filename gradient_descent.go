package main

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
	scaleSample := func(x sample) sample {
		var underExists, overExists bool
		var underSample, overSample sample
		if x.PPIter.ProbResult < TargetProbability {
			underExists = true
			underSample = x
		} else {
			overExists = true
			overSample = x
		}
		for range 100 {
			if overExists && overSample.PPIter.ProbResult < TargetProbability+1e-5 {
				return overSample
			}
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
			if nextSample.PPIter.ProbResult < TargetProbability {
				underExists = true
				underSample = nextSample
			} else {
				overExists = true
				overSample = nextSample
			}
		}
		panic("skills didn't converge")
	}

	var ret sample
	{
		for i := range skillCount {
			ret.SkillVector[i] = 300
		}
		ret.PPIter = fn(VectorToSkills(ret.SkillVector))
		ret = scaleSample(ret)
	}

	maxDelta := 1.0
	for _, skill := range ret.SkillVector {
		maxDelta = max(maxDelta, skill*2)
	}

	for delta := maxDelta; delta >= 0.01; delta /= 2 {
		improved := true
		for improved {
			improved = false
			for _, sign := range []float64{-1, 1} {
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
