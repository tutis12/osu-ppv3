package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

func ScrapeBeatmaps(toScrape []int) {
	wg := sync.WaitGroup{}
	for index := 0; index < len(toScrape); index += 50 {
		wg.Add(1)
		Run(func() {
			defer wg.Done()
			ids := make([]int, min(50, len(toScrape)-index))
			for i := range len(ids) {
				ids[i] = toScrape[i] + i
			}
			beatmaps, err := FetchBeatmaps(context.Background(), ids)
			if err != nil {
				panic(err.Error())
			}
			for _, beatmap := range beatmaps {
				file, err := os.Create(fmt.Sprintf("../_beatmaps/%d", beatmap.ID))
				if err != nil {
					panic(err.Error())
				}
				data, err := json.Marshal(beatmap)
				if err != nil {
					panic(err.Error())
				}
				_, err = file.Write(data)
				if err != nil {
					panic(err.Error())
				}
			}
		})
	}
	wg.Wait()
}
