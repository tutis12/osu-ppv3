package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

func ScrapeBeatmaps() {
	wg := sync.WaitGroup{}
	total := atomic.Uint64{}
	for id := 761100; id < 5_100_000; id += 50 {
		wg.Add(1)
		Run(func() {
			defer wg.Done()
			ids := make([]int, 50)
			for i := range 50 {
				ids[i] = id + i
			}
			beatmaps, err := FetchBeatmaps(context.Background(), ids)
			total.Add(50)
			fmt.Println(id, "...", id+49, "/", total.Load())
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
