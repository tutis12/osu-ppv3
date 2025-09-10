package main

import (
	"fmt"
	"math"
	"ppv3/dotosu"
)

type BeatmapPPInfo struct {
	Iter PPIter
}

type BeatmapIdentifier struct {
	Id   int
	Mods Modifiers
}

func CalculateBeatmapPPInfo(
	beatmap *dotosu.Beatmap,
	mapConstants MapConstants,
	actions []*Action,
	count100s int,
	count50s int,
	countMisses int,
	countSliderEndMisses int,
	countSliderTickMisses int,
	countSpinnerMisses int,
) (*BeatmapPPInfo, error) {
	fmt.Println(beatmap.Metadata.Title)
	ppIter := GradientDescent(
		func(skills Skills) PPIter {
			iter := NewPPIter(
				mapConstants,
				skills,
			)
			for _, action := range actions {
				IterateAction(&iter, action)
			}
			iter.CalculateProbability(
				count100s,
				count50s,
				countMisses,
				countSliderEndMisses,
				countSliderTickMisses,
				countSpinnerMisses,
			)
			iter.PP = skills.PP()

			return iter
		},
	)

	mods := mapConstants.Mods

	modsStr := ""
	if mods.Rate > 1 {
		modsStr += fmt.Sprintf("DT(%.2f)", mods.Rate)
	}
	if mods.Rate < 1 {
		modsStr += fmt.Sprintf("HT(%.2f)", mods.Rate)
	}
	if mods.Easy {
		modsStr += "EZ"
	}
	if mods.Hardrock {
		modsStr += "HR"
	}
	if mods.Hidden {
		modsStr += "HD"
	}
	if mods.Flashlight {
		modsStr += "FL"
	}
	if mods.NoFail {
		modsStr += "NF"
	}
	if mods.SpunOut {
		modsStr += "SO"
	}

	if modsStr == "" {
		modsStr = "NM"
	}

	fmt.Printf(
		"%s [%s]\n%s\n%d x 100s\n%d x 50s\n%d x misses \n%d x slider end misses\n%d x slider tick misses\n%d x spinner misses\n%.5fpp\n\n",
		beatmap.Metadata.Title, beatmap.Metadata.Version,
		modsStr,
		count100s, count50s, countMisses,
		countSliderEndMisses,
		countSliderTickMisses,
		countSpinnerMisses,
		ppIter.PP,
	)

	return &BeatmapPPInfo{
		Iter: ppIter,
	}, nil
}

func ApproachRateToPreempt(ar float64) float64 {
	if ar < 5 {
		return 1200 + 120*(5-ar)
	} else if ar == 5 {
		return 1200
	} else {
		return 1200 - 150*(ar-5)
	}
}

func PreemptToAR(preempt float64) float64 {
	if preempt > 1200 {
		return 5 - (preempt-1200)/120
	} else if preempt == 1200 {
		return 5
	} else {
		return 5 + (1200-preempt)/150
	}
}

type Vec struct {
	X, Y float64
}

func Distance(a, b Vec) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

var CenterPos = Vec{
	X: 256,
	Y: 192,
}
