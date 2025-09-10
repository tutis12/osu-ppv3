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
	users := []int{10077431, 7562902, 17592067}
	for _, userId := range users {
		pprecalc, err := EvalUserScores(userId)
		if err != nil {
			panic(err)
		}

		file, err := os.Create(fmt.Sprintf("user_%d.txt", userId))
		if err != nil {
			panic(err)
		}
		json, _ := json.MarshalIndent(pprecalc, "", "\t")
		_, err = file.Write(json)
		if err != nil {
			panic(err)
		}
		file.Close()
	}
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

func OpenBeatmap(id int) (*Beatmap, *dotosu.Beatmap) {
	info := LoadBeatmap(id)

	{
		dir := fmt.Sprintf("../_ranked_sets/%d", info.BeatmapsetID)
		_, err := os.Stat(dir)
		if err != nil {
			DownloadSets([]Beatmapset{info.Beatmapset})
		}
	}
	set, allFiles, err := OpenSet(info.BeatmapsetID)
	if err != nil {
		panic(err)
	}
	var ids []int
	for _, m := range set {
		ids = append(ids, m.Metadata.BeatmapID)
		if m.Metadata.BeatmapID == id || m.Metadata.Version == info.Version {
			return info, m
		}
	}
	panic(fmt.Sprintf("not found %d %d %v %v", id, info.BeatmapsetID, allFiles, ids))
}

func OpenSet(id int) ([]*dotosu.Beatmap, []string, error) {
	var allFiles []string
	dir := fmt.Sprintf("../_ranked_sets/%d", id)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, allFiles, err
	}
	if !info.IsDir() {
		return nil, allFiles, fmt.Errorf("path is not a directory: %s", dir)
	}

	var paths []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		allFiles = append(allFiles, d.Name())
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".osu") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, allFiles, err
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
		return beatmaps, allFiles, fmt.Errorf("decoded %d/%d .osu files; first failure %s: %w", len(beatmaps), len(paths), firstErrPath, firstErr)
	}
	return beatmaps, allFiles, nil
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
		beatmap := LoadBeatmap(id)
		beatmaps = append(beatmaps, beatmap)
	}
	return beatmaps
}

func LoadBeatmap(id int) *Beatmap {
	file, err := os.Open(fmt.Sprintf("../_beatmaps/%d", id))
	if err != nil {
		ScrapeBeatmaps([]int{id})
		file, err = os.Open(fmt.Sprintf("../_beatmaps/%d", id))
		if err != nil {
			panic(err)
		}
	}
	var beatmap Beatmap
	bytes, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &beatmap)
	if err != nil {
		panic(err)
	}
	return &beatmap
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
