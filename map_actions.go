package main

import (
	"encoding/json"
	"fmt"
	"math"
	"ppv3/dotosu"
)

type Action struct {
	Pos        Vec
	Time       float64
	Radius     float64 //circle/sliderhead < sliderend < spinner
	Clickable  bool
	Circle     bool
	SliderEnd  bool
	SliderTick bool
	Spinner    bool

	LastClicks []TimePos
	LastAims   []TimePos
}

func ConvertBeatmapToActions(beatmap *dotosu.Beatmap, mods Modifiers) ([]*Action, error) {
	actions := make([]*Action, 0, len(beatmap.HitObjects))

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
		for timingPointIndex < len(timingPoints) && (lastRedLine == nil || timingPoints[timingPointIndex].Time <= object.StartTime()) {
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
				&Action{
					Pos: Vec{
						X: float64(object.PosXY.X),
						Y: float64(object.PosXY.Y),
					},
					Time:      float64(object.Time),
					Radius:    csInPixels,
					Clickable: true,
					Circle:    true,
				},
			)
		case dotosu.Slider:
			beatLength := lastRedLine.BeatLength
			var sv float64
			if lastGreenLine != nil {
				sv = max(0.1, lastGreenLine.SliderVelocityMultiplier)
			} else {
				sv = 1
			}

			samples := ApproximateSliderPath(object)

			actions = append(
				actions,
				&Action{
					Pos: Vec{
						X: float64(object.PosXY.X),
						Y: float64(object.PosXY.Y),
					},
					Time:      float64(object.Time),
					Radius:    csInPixels,
					Clickable: true,
				},
			)
			visualLength := object.Length
			timeLength := visualLength / (beatmap.Difficulty.SliderMultiplier * 100 * sv) * beatLength

			ticksFloat := timeLength / beatLength * beatmap.Difficulty.SliderTickRate
			ticks := max(0, int(math.Floor((timeLength-min(36, timeLength/2))/beatLength*beatmap.Difficulty.SliderTickRate)))
			tickLength := visualLength / ticksFloat

			tickTime := beatLength / beatmap.Difficulty.SliderTickRate

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
						&Action{
							Pos:        GetSliderPosition(samples, progress),
							Time:       time,
							Radius:     csInPixels * 2.4,
							Clickable:  false,
							SliderTick: true,
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
					&Action{
						Pos:        sliderend,
						Time:       repeatTime,
						Radius:     csInPixels * 2.4,
						Clickable:  false,
						SliderEnd:  i == object.Slides-1,
						SliderTick: i != object.Slides-1,
					},
				)
			}
		case dotosu.Spinner:
			if mods.SpunOut {
				continue objectLoop
			}
			actions = append(
				actions,
				&Action{
					Pos:       CenterPos,
					Time:      float64(object.Time+object.EndTime) / 2,
					Radius:    200, // should be based on od
					Clickable: false,
					Spinner:   true,
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
			return nil, fmt.Errorf("actions too close at times:\n%v\n%v\nbeatmapId=%d\n", string(a), string(b), beatmap.Metadata.BeatmapID)
		}
	}
	for i := range actions {
		actions[i].Time /= mods.Rate
	}

	clicks := make([]TimePos, fakeObjects)
	aims := make([]TimePos, fakeObjects)
	for i := range fakeObjects {
		fakeObject := TimePos{
			Pos:    CenterPos,
			Radius: 1000,
			Time:   -1e18 + 1e12*float64(i),
		}
		clicks[i] = fakeObject
		aims[i] = fakeObject
	}

	for i, action := range actions {
		actions[i].LastClicks = clicks
		actions[i].LastAims = aims
		timePos := TimePos{
			Pos:    action.Pos,
			Radius: action.Radius,
			Time:   action.Time,
		}
		if action.Clickable {
			clicks = append(
				clicks,
				timePos,
			)
		}
		aims = append(aims, timePos)
	}

	return actions, nil
}
