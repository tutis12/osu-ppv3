package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"
)

type Play struct {
	BeatmapID  int
	Artist     string
	Title      string
	Difficulty string
	StarRating float64

	PrevPP     float64
	NewPP      float64
	Weight     float64
	WeightedPP float64

	Skills   Skills
	OldIndex int
	NewIndex int

	similaritySum float64
}

func EvalUserScores(userId int) ([]*Play, error) {
	scores, err := GetBestScores(userId, 500)
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(scores))
	recalc := make([]*Play, len(scores))
	for i, score := range scores {
		Run(func() {
			defer wg.Done()
			mods := Modifiers{
				Lazer: score.Score == 0,

				Rate:       1.0,
				Hardrock:   slices.Contains(score.Mods, "HR"),
				Easy:       slices.Contains(score.Mods, "EZ"),
				Hidden:     slices.Contains(score.Mods, "HD"),
				Flashlight: slices.Contains(score.Mods, "FL"),
				NoFail:     slices.Contains(score.Mods, "NF"),
				SpunOut:    slices.Contains(score.Mods, "SO"),
			}
			if slices.Contains(score.Mods, "DT") {
				mods.Rate = 1.5
			}
			if slices.Contains(score.Mods, "HT") {
				mods.Rate = 0.75
			}

			calculate := CalculateScore(
				score.Beatmap.ID,
				mods,
				score.Statistics.Count100,
				score.Statistics.Count50,
				score.Statistics.CountMiss,
			)
			recalc[i] = &Play{
				BeatmapID:  score.Beatmap.BeatmapsetID,
				Artist:     score.BeatmapSet.Artist,
				Title:      score.BeatmapSet.Title,
				Difficulty: score.Beatmap.Version,
				StarRating: score.Beatmap.DifficultyRating,
				PrevPP:     score.PP,
				NewPP:      calculate.Iter.PP,
				Skills:     calculate.Iter.Skills,
				OldIndex:   i,
			}
			fmt.Println(i, score.BeatmapSet.Title)
		})
	}
	wg.Wait()
	ret := make([]*Play, 0, len(recalc))
	for range len(recalc) {
		for _, score := range recalc {
			score.Weight = math.Pow(0.95, 0.1*float64(len(ret))+0.9*score.similaritySum)
			score.WeightedPP = score.NewPP * score.Weight
		}
		slices.SortFunc(recalc, func(a, b *Play) int {
			return cmp.Compare(a.WeightedPP, b.WeightedPP)
		})
		max := recalc[len(recalc)-1]
		recalc = recalc[:len(recalc)-1]
		max.NewIndex = len(ret)
		ret = append(ret, max)
		for _, score := range recalc {
			score.similaritySum += Similarity(score.Skills, max.Skills)
		}
	}
	return ret, nil
}

func Similarity(aSkills, bSkills Skills) float64 {
	a, b := SkillsToVector(aSkills), SkillsToVector(bSkills)
	dotProduct := 0.0
	aSquare := 0.0
	bSquare := 0.0
	for i := range skillCount {
		dotProduct += a[i] * b[i]
		aSquare += a[i] * a[i]
		bSquare += b[i] * b[i]
	}
	return dotProduct / math.Sqrt(aSquare*bSquare)
}

func CalculateScore(
	beatmapId int,
	mods Modifiers,
	count100s int,
	count50s int,
	countMisses int,
) *BeatmapPPInfo {
	_, beatmap := OpenBeatmap(beatmapId)
	actions, err := ConvertBeatmapToActions(beatmap, mods)
	if err != nil {
		panic(err)
	}

	var hits, sliderends, sliderticks int
	for _, action := range actions {
		if action.Clickable {
			hits++
		} else if action.SliderEnd {
			sliderends++
		} else if action.SliderTick {
			sliderticks++
		}
	}
	info, err := CalculateBeatmapPPInfo(
		beatmap,
		actions,
		mods,
		count100s,
		count50s,
		countMisses,
		0,
		0,
		0,
	)
	if err != nil {
		panic(err.Error())
	}
	return info
}
