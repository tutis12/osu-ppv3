package main

import "ppv3/dotosu"

type MapConstants struct {
	Mods         Modifiers
	CircleRadius float64
	ApproachRate float64
	Preempt      float64
	Window300    float64
	Window100    float64
	Window50     float64
}

func GetBeatmapConstants(
	beatmap *dotosu.Beatmap,
	mods Modifiers,
) MapConstants {
	cs := beatmap.Difficulty.CircleSize
	if mods.Hardrock {
		cs = min(cs*1.3, 10)
	}
	if mods.Easy {
		cs = cs / 2
	}

	circleRadius := 54.4 - 4.48*cs

	ar := beatmap.Difficulty.ApproachRate

	od := beatmap.Difficulty.OverallDifficulty
	if mods.Hardrock {
		od = min(10, od*1.4)
	}
	if mods.Easy {
		od = od / 2
	}

	if mods.Hardrock {
		ar = min(10, ar*1.4)
	}
	if mods.Easy {
		ar = ar / 2
	}

	preempt := ApproachRateToPreempt(ar) / mods.Rate
	ar = PreemptToAR(preempt)

	window300 := (80 - 6*od) / mods.Rate //+- this
	window100 := (140 - 8*od) / mods.Rate
	window50 := (200 - 10*od) / mods.Rate

	return MapConstants{
		Mods:         mods,
		CircleRadius: circleRadius,
		ApproachRate: ar,
		Preempt:      preempt,
		Window300:    window300,
		Window100:    window100,
		Window50:     window50,
	}
}
