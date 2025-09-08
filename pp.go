package main

import (
	"encoding/json"
	"fmt"
	"math"
	"ppv3/dotosu"
)

type BeatmapPPInfo struct {
	Actions   []Action
	Window300 float64
	Window100 float64
	Window50  float64
}

type Skills struct {
	Aim     AimSkills
	Tapping TappingSkills
}

func VectorToSkills(skillVector [36]float64) Skills {
	return Skills{
		Aim: AimSkills{
			Precision:        skillVector[0],
			Speed:            skillVector[1],
			Stamina:          skillVector[2],
			SpeedIncrease:    skillVector[3],
			SpeedDecrease:    skillVector[4],
			SpeedStop:        skillVector[5],
			FlicksWideAngle:  skillVector[6],
			FlicksSharpAngle: skillVector[7],
			FlicksRightAngle: skillVector[8],
			Cheese:           skillVector[9],
			AngleIncrease:    skillVector[10],
			AngleDecrease:    skillVector[11],
			CW:               skillVector[12],
			CCW:              skillVector[13],
			CWChange:         skillVector[14],
			CWHold:           skillVector[15],
			Angles:           [18]float64(skillVector[16:34]),
		},
		Tapping: TappingSkills{
			Accuracy: skillVector[34],
			Speed:    skillVector[35],
		},
	}
}

func SkillsToVector(skills Skills) [36]float64 {
	return [36]float64{
		skills.Aim.Precision,
		skills.Aim.Speed,
		skills.Aim.Stamina,
		skills.Aim.SpeedIncrease,
		skills.Aim.SpeedDecrease,
		skills.Aim.SpeedStop,
		skills.Aim.FlicksWideAngle,
		skills.Aim.FlicksSharpAngle,
		skills.Aim.FlicksRightAngle,
		skills.Aim.Cheese,
		skills.Aim.AngleIncrease,
		skills.Aim.AngleDecrease,
		skills.Aim.CW,
		skills.Aim.CCW,
		skills.Aim.CWChange,
		skills.Aim.CWHold,
		skills.Aim.Angles[0],
		skills.Aim.Angles[1],
		skills.Aim.Angles[2],
		skills.Aim.Angles[3],
		skills.Aim.Angles[4],
		skills.Aim.Angles[5],
		skills.Aim.Angles[6],
		skills.Aim.Angles[7],
		skills.Aim.Angles[8],
		skills.Aim.Angles[9],
		skills.Aim.Angles[10],
		skills.Aim.Angles[11],
		skills.Aim.Angles[12],
		skills.Aim.Angles[13],
		skills.Aim.Angles[14],
		skills.Aim.Angles[15],
		skills.Aim.Angles[16],
		skills.Aim.Angles[17],
		skills.Tapping.Accuracy,
		skills.Tapping.Speed,
	}
}

type AimSkills struct {
	Precision float64 // high cs
	Speed     float64 // high bpm
	Stamina   float64 // long circle chains

	SpeedIncrease float64 // increase cursor speed
	SpeedDecrease float64 // decrease cursor speed
	SpeedStop     float64 // hold still / stacks / sliders

	FlicksWideAngle  float64 // wide angle flicks
	FlicksSharpAngle float64 // sharp angle flicks
	FlicksRightAngle float64 // right angle flicks

	Cheese float64 // cheese overlaps

	AngleIncrease float64 // increase jump angle
	AngleDecrease float64 // decrease jump angle

	CW  float64 // clockwise flow
	CCW float64 // counterclockwise flow

	CWChange float64     // ability to switch circular direction
	CWHold   float64     // ability to continue circular direction
	Angles   [18]float64 // increments of 10 degrees
}

type TappingSkills struct {
	Accuracy float64 // high od
	Speed    float64 // high bpm
}

func CalculateBeatmapPPInfo(beatmapID int, mods Modifiers) (*BeatmapPPInfo, error) {
	_, beatmap, err := OpenBeatmap(beatmapID)
	if err != nil {
		return nil, err
	}
	actions, err := ConvertBeatmapToActions(beatmap, mods)
	if err != nil {
		return nil, err
	}

	od := beatmap.Difficulty.OverallDifficulty
	if mods.Hardrock {
		od = min(10, od*1.4)
	}
	if mods.Easy {
		od = od / 2
	}

	window300 := (80 - 6*od) / mods.Rate //+- this
	window100 := (140 - 8*od) / mods.Rate
	window50 := (200 - 10*od) / mods.Rate

	return &BeatmapPPInfo{
		Actions:   actions,
		Window300: window300,
		Window100: window100,
		Window50:  window50,
	}, nil
}

type Vec struct {
	X, Y float64
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
					if j%2 == 0 {
						progress = float64(j+1) * tickLength
					} else {
						progress = float64(ticks-j) * tickLength
					}
					actions = append(
						actions,
						Action{
							Pos: GetSliderPosition(samples, progress),
							Time: float64(object.Time) +
								float64(i)*timeLength +
								float64(j+1)*tickTime,
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
					effectiveLength := min(36, timeLength/2)
					repeatTime -= effectiveLength
					var progress float64
					if i%2 == 0 {
						progress = effectiveLength / timeLength * visualLength
					} else {
						progress = (1 - effectiveLength/timeLength) * visualLength
					}
					sliderend = GetSliderPosition(samples, progress)
				} else {
					if i%2 == 0 {
						sliderend = samples[len(samples)-1]
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
					Pos: Vec{
						X: 256,
						Y: 192,
					},
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
