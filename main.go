package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"ppv3/dotosu"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	userId := 17592067
	_, err := EvalUserScores(userId)
	if err != nil {
		panic(err)
	}
}

type PPRecalc struct {
	BeatmapID  int
	Artist     string
	Title      string
	Difficulty string
	PrevPP     float64
	NewPP      float64
	Info       *BeatmapPPInfo
}

func EvalUserScores(userId int) ([]PPRecalc, error) {
	scores, err := GetBestScores(userId)
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(scores))
	recalc := make([]PPRecalc, len(scores))
	for i, score := range scores {
		Run(func() {
			defer wg.Done()
			mods := Modifiers{
				Lazer: true,

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
				score.Beatmap.BeatmapsetID,
				mods,
				score.Statistics.Count100,
				score.Statistics.Count50,
				score.Statistics.CountMiss,
				0,
				0,
				0,
			)
			recalc[i] = PPRecalc{
				BeatmapID:  score.Beatmap.BeatmapsetID,
				Artist:     score.BeatmapSet.Artist,
				Title:      score.BeatmapSet.Title,
				Difficulty: score.Beatmap.Version,
				PrevPP:     score.PP,
				NewPP:      calculate.Iter.PP,
				Info:       calculate,
			}
			json, _ := json.MarshalIndent(recalc[i], "", "\t")
			fmt.Println(string(json))
		})
	}
	wg.Wait()
	return recalc, nil
}

func CalculateScore(
	beatmapId int,
	mods Modifiers,
	count100s int,
	count50s int,
	countMisses int,
	countSliderEndMisses int,
	countSliderTickMisses int,
	countSpinnerMisses int,
) *BeatmapPPInfo {
	_, beatmap, err := OpenBeatmap(beatmapId)
	if err != nil {
		panic(err)
	}
	actions, err := ConvertBeatmapToActions(beatmap, mods)
	if err != nil {
		panic(err)
	}

	info, err := CalculateBeatmapPPInfo(
		beatmap,
		actions,
		mods,
		count100s,
		count50s,
		countMisses,
		countSliderEndMisses,
		countSliderTickMisses,
		countSpinnerMisses,
	)
	if err != nil {
		panic(err.Error())
	}
	return info
}

type Modifiers struct {
	Lazer bool

	Rate float64

	Hardrock bool
	Easy     bool

	Hidden     bool
	Flashlight bool

	NoFail  bool
	SpunOut bool
}

func OpenBeatmap(id int) (*Beatmap, *dotosu.Beatmap, error) {
	info, err := LoadBeatmap(id)
	if err != nil {
		return nil, nil, err
	}

	_, err = DownloadBeatmapset(info.Beatmapset)
	if err != nil {
		panic(err)
	}
	{
		dir := fmt.Sprintf("../_ranked_sets/%d", info.BeatmapsetID)
		_, err := os.Stat(dir)
		if err != nil {
			_, err := DownloadBeatmapset(info.Beatmapset)
			if err != nil {
				panic(err)
			}
		}
	}
	set, err := OpenSet(info.BeatmapsetID)
	if err != nil {
		return nil, nil, err
	}
	for _, m := range set {
		if m.Metadata.BeatmapID == id {
			return info, m, nil
		}
	}
	return nil, nil, fmt.Errorf("not found")
}

func OpenSet(id int) ([]*dotosu.Beatmap, error) {
	dir := fmt.Sprintf("../_ranked_sets/%d", id)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	var paths []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".osu") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(paths)

	beatmaps := make([]*dotosu.Beatmap, 0, len(paths))
	var firstErr error
	var firstErrPath string
	failCount := 0

	for _, p := range paths {
		bm, e := dotosu.DecodeFile(p)
		if e != nil {
			failCount++
			if firstErr == nil {
				firstErr = e
				firstErrPath = p
			}
			continue
		}
		beatmaps = append(beatmaps, bm)
	}

	if firstErr != nil {
		return beatmaps, fmt.Errorf("decoded %d/%d .osu files; first failure %s: %w", len(beatmaps), len(paths), firstErrPath, firstErr)
	}
	return beatmaps, nil
}

func DownloadSets(rankedSets []Beatmapset) {
	wg := sync.WaitGroup{}
	counter := atomic.Uint32{}
	slices.Reverse(rankedSets)
	total := atomic.Uint32{}
	for _, set := range rankedSets {
		if set.Availability.DownloadDisabled {
			bytes, _ := json.Marshal(set)
			Fail("_skips", set.ID, string(bytes))
			continue
		}
		{
			file, err := os.Open(fmt.Sprintf("../_ranked_sets/%d", set.ID))
			if err == nil {
				file.Close()
				foundDotOsu := false
				err = filepath.Walk(fmt.Sprintf("../_ranked_sets/%d", set.ID), func(path string, info os.FileInfo, err error) error {
					if err != nil {
						panic(err)
					}
					if strings.HasSuffix(info.Name(), ".osu") {
						foundDotOsu = true
					}

					return nil
				})
				if err != nil {
					panic(err)
				}
				if foundDotOsu {
					fmt.Printf("%d already downloaded (%s)\n", set.ID, set.Title)
					continue
				}
			}
		}
		wg.Add(1)
		Run(func() {
			defer wg.Done()
			total.Add(1)
			files, err := DownloadBeatmapset(set)
			if err != nil {
				PanicF("DownloadBeatmapset failed id = %d, err = %s", set.ID, err.Error())
			}
			err = os.Mkdir(fmt.Sprintf("../_ranked_sets/%d", set.ID), 0777)
			if err != nil && !strings.Contains(err.Error(), "file exists") {
				PanicF("Mkdir failed id = %d, err = %s", set.ID, err.Error())
			}
			counter.Add(1)
			fmt.Printf("%d downloaded (%s)\n", set.ID, set.Title)
			fmt.Printf("%d/%d\n\n", counter.Load(), total.Load())
			for fileName, data := range files {
				file, err := os.Create(fmt.Sprintf("../_ranked_sets/%d/%s", set.ID, fileName))
				if err != nil {
					PanicF("os.Create failed id = %d, err = %s", set.ID, err.Error())
				}
				defer file.Close()
				_, err = file.Write(data)
				if err != nil {
					PanicF("file.Write failed id = %d, err = %s", set.ID, err.Error())
				}
			}
		})
	}
	wg.Wait()
}

func RankedOsuBeatmapsets() []Beatmapset {
	beatmaps := LoadBeatmaps()
	beatmapsByMode := make(map[string][]*Beatmap)
	for _, beatmap := range beatmaps {
		if beatmap.Status != "ranked" {
			continue
		}
		beatmapsByMode[beatmap.Mode] = append(
			beatmapsByMode[beatmap.Mode],
			beatmap,
		)
	}
	osuRankedBeatmaps := beatmapsByMode["osu"]
	osuRankedBeatmapsets := make(map[int][]*Beatmap)
	for _, beatmap := range osuRankedBeatmaps {
		osuRankedBeatmapsets[beatmap.BeatmapsetID] = append(
			osuRankedBeatmapsets[beatmap.BeatmapsetID],
			beatmap,
		)
	}
	sets := make([]Beatmapset, 0, len(osuRankedBeatmapsets))
	for _, maps := range osuRankedBeatmapsets {
		sets = append(sets, maps[0].Beatmapset)
	}
	slices.SortFunc(sets, func(i, j Beatmapset) int {
		return cmp.Compare(i.ID, j.ID)
	})
	fmt.Printf("found %d ranked osu sets\n\n", len(sets))
	return sets
}

func LoadBeatmaps() []*Beatmap {
	var beatmaps []*Beatmap
	ids := loadBeatmapIDS()
	for _, id := range ids {
		beatmap, err := LoadBeatmap(id)
		if err != nil {
			panic(err)
		}
		beatmaps = append(beatmaps, beatmap)
	}
	return beatmaps
}

func LoadBeatmap(id int) (*Beatmap, error) {
	file, err := os.Open(fmt.Sprintf("../_beatmaps/%d", id))
	if err != nil {
		ScrapeBeatmaps([]int{id})
		file, err = os.Open(fmt.Sprintf("../_beatmaps/%d", id))
		if err != nil {
			return nil, err
		}
	}
	var beatmap Beatmap
	bytes, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &beatmap)
	if err != nil {
		return nil, err
	}
	return &beatmap, nil
}

func loadBeatmapIDS() []int {
	ids := make([]int, 0)
	err := filepath.Walk("../_beatmaps", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		if !info.IsDir() {
			var id int
			_, err := fmt.Sscanf(info.Name(), "%d", &id)
			if err != nil {
				panic(err)
			}
			ids = append(ids, id)
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
	return ids
}
