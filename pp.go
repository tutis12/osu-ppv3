package main

import (
	"encoding/json"
	"fmt"
	"math"
	"ppv3/dotosu"
)

type BeatmapPPInfo struct {
	Iter              PPIter
	ApproachRate      float64
	OverallDifficulty float64
	Actions           []Action
}

func CalculateBeatmapPPInfo(
	beatmapID int,
	mods Modifiers,
	count100s int,
	count50s int,
	countMisses int,
) (*BeatmapPPInfo, error) {
	_, beatmap, err := OpenBeatmap(beatmapID)
	if err != nil {
		return nil, err
	}
	actions, err := ConvertBeatmapToActions(beatmap, mods)
	if err != nil {
		return nil, err
	}

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

	od = (80 - window300) / 6

	ppIter := GradientDescent(
		func(skills Skills) PPIter {
			iter := NewPPIter(
				skills,
				window300,
				window100,
				window50,
			)
			for _, action := range actions {
				IterateAction(&iter, &action)
			}
			iter.CalculateProbability(
				count100s,
				count50s,
				countMisses,
			)
			iter.PP = skills.PP()
			return iter
		},
	)

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

	fmt.Printf("%s [%s]\n%s\n%dx100s %dx50s %dxmisses\n%.5fpp\n\n", beatmap.Metadata.Title, beatmap.Metadata.Version, modsStr, count100s, count50s, countMisses, ppIter.PP)

	return &BeatmapPPInfo{
		Iter:              ppIter,
		ApproachRate:      ar,
		OverallDifficulty: od,
		Actions:           actions,
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

var CenterPos = Vec{
	X: 256,
	Y: 192,
}

type Action struct {
	Pos        Vec
	Time       float64
	Radius     float64 //circle/sliderhead < sliderend < spinner
	Clickable  bool
	Object     dotosu.HitObject
	SliderPath []Vec `json:"SliderPath,omitzero"`
}

func ConvertBeatmapToActions(beatmap *dotosu.Beatmap, mods Modifiers) ([]Action, error) {
	actions := make([]Action, 0, len(beatmap.HitObjects))

	cs := beatmap.Difficulty.CircleSize
	if mods.Hardrock {
		cs = min(cs*1.3, 10)
	}
	if mods.Easy {
		cs = cs / 2
	}

	csInPixels := 54.4 - 4.48*cs

	timingPoints := beatmap.TimingPoints
	timingPointIndex := 0
	var lastRedLine *dotosu.TimingPoint
	var lastGreenLine *dotosu.TimingPoint
objectLoop:
	for _, object := range beatmap.HitObjects {
		for timingPointIndex < len(timingPoints) && timingPoints[timingPointIndex].Time <= object.StartTime() {
			timingPoint := timingPoints[timingPointIndex]
			timingPointIndex++

			if timingPoint.TimingChange {
				lastRedLine = &timingPoint
				lastGreenLine = nil
			} else {
				lastGreenLine = &timingPoint
			}
		}
		switch object := object.(type) {
		case dotosu.Circle:
			actions = append(
				actions,
				Action{
					Pos: Vec{
						X: float64(object.PosXY.X),
						Y: float64(object.PosXY.Y),
					},
					Time:      float64(object.Time),
					Radius:    csInPixels,
					Clickable: true,
					Object:    object,
				},
			)
		case dotosu.Slider:
			beatLength := lastRedLine.BeatLength
			var sv float64
			if lastGreenLine != nil {
				sv = lastGreenLine.SliderVelocityMultiplier
			} else {
				sv = 1
			}

			samples := ApproximateSliderPath(object)

			actions = append(
				actions,
				Action{
					Pos: Vec{
						X: float64(object.PosXY.X),
						Y: float64(object.PosXY.Y),
					},
					Time:       float64(object.Time),
					Radius:     csInPixels,
					Clickable:  true,
					Object:     object,
					SliderPath: samples,
				},
			)
			visualLength := object.Length
			timeLength := visualLength / (beatmap.Difficulty.SliderMultiplier * 100 * sv) * beatLength

			ticksFloat := timeLength / beatLength * beatmap.Difficulty.SliderTickRate
			ticks := max(0, int(math.Floor(ticksFloat-1e-5)))
			tickLength := visualLength / ticksFloat

			tickTime := beatLength / beatmap.Difficulty.SliderTickRate

			if object.Time == 1100 {
				fmt.Printf("beatLength: %f\n", beatLength)
				fmt.Printf("sv: %f\n", sv)
				fmt.Printf("visualLength: %f\n", visualLength)
				fmt.Printf("timeLength: %f\n", timeLength)
				fmt.Printf("ticksFloat: %f\n", ticksFloat)
				fmt.Printf("ticks: %d\n", ticks)
				fmt.Printf("tickLength: %f\n", tickLength)
				fmt.Printf("tickTime: %f\n\n", tickTime)
			}

			for i := range object.Slides {
				for j := range ticks {
					var progress float64
					var time float64
					if i%2 == 0 {
						time = float64(object.Time) +
							float64(i)*timeLength +
							float64(j+1)*tickTime
						progress = float64(j+1) * tickLength
					} else {
						time = float64(object.Time) +
							float64(i+1)*timeLength +
							float64(j-ticks)*tickTime
						progress = float64(ticks-j) * tickLength
					}
					actions = append(
						actions,
						Action{
							Pos:        GetSliderPosition(samples, progress),
							Time:       time,
							Radius:     csInPixels * 2.4,
							Clickable:  false,
							Object:     object,
							SliderPath: samples,
						},
					)
				}
				repeatTime := float64(object.Time) +
					float64(i+1)*timeLength
				var sliderend Vec
				if i == object.Slides-1 {
					effectiveLength := timeLength - min(36, timeLength/2)
					repeatTime -= min(36, timeLength/2)
					var progress float64
					if i%2 == 0 {
						progress = effectiveLength / timeLength * visualLength
					} else {
						progress = (1 - effectiveLength/timeLength) * visualLength
					}
					sliderend = GetSliderPosition(samples, progress)
				} else {
					if i%2 == 0 {
						sliderend = GetSliderPosition(samples, visualLength)
					} else {
						sliderend = Vec{
							X: float64(object.PosXY.X),
							Y: float64(object.PosXY.Y),
						}
					}
				}
				actions = append(
					actions,
					Action{
						Pos:        sliderend,
						Time:       repeatTime,
						Radius:     csInPixels * 2.4,
						Clickable:  false,
						Object:     object,
						SliderPath: samples,
					},
				)
			}
		case dotosu.Spinner:
			if mods.SpunOut {
				continue objectLoop
			}
			actions = append(
				actions,
				Action{
					Pos:       CenterPos,
					Time:      float64(object.Time+object.EndTime) / 2,
					Radius:    100, // should be based on od
					Clickable: false,
					Object:    object,
				},
			)
		default:
			panic("unexpected")
		}
	}
	for i := 1; i < len(actions); i++ {
		if actions[i-1].Time >= actions[i].Time {
			a, _ := json.Marshal(actions[i-1])
			b, _ := json.Marshal(actions[i])
			return nil, fmt.Errorf("actions too close at times:\n%v\n%v\n", string(a), string(b))
		}
	}
	for i := range actions {
		actions[i].Time /= mods.Rate
	}

	return actions, nil
}
