package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var rateLimitedFrom atomic.Pointer[time.Time]

func rateLimited() time.Duration {
	lastLimit := rateLimitedFrom.Load()
	now := time.Now()
	rateLimitedFrom.CompareAndSwap(nil, &now)
	if lastLimit != nil {
		return max(time.Minute, time.Since(*lastLimit))
	}
	return time.Minute
}

func DownloadBeatmapset(set Beatmapset) (map[string][]byte, error) {
	done := GetToken()
	defer done()
	var bytes []byte
	for {
		var err error
		bytes, err = DownloadBeatmapsetBytes(set.ID)
		if err != nil && strings.Contains(err.Error(), "connection refused") {
			cooldown := rateLimited()
			fmt.Printf("\n\n\n\n\nconnection refused, %s\n", cooldown)
			time.Sleep(max(time.Minute, cooldown))
			continue
		}
		if strings.Contains(string(bytes), "Slow down, play more.") {
			cooldown := rateLimited()
			fmt.Printf("\n\n\n\n\nSlow down, play more. %s\n", cooldown)
			time.Sleep(max(time.Minute, cooldown))
			continue
		}
		rateLimitedFrom.Store(nil)
		if err == nil {
			break
		} else {
			fmt.Printf("DownloadBeatmapsetBytes failed id = %d, err = %s", set.ID, err.Error())
			time.Sleep(time.Minute)
			continue
		}
	}
	// Treat the .osz file as a ZIP file by reading it using zip.NewReader
	zipReader, err := zip.NewReader(strings.NewReader(string(bytes)), int64(len(bytes)))
	if err != nil {
		if strings.Contains(err.Error(), "not a valid zip file") {
			bytes, _ := json.Marshal(set)
			Fail("_not_zip", set.ID, err.Error()+"\n\n"+string(bytes))
			return map[string][]byte{}, nil
		}
		return nil, fmt.Errorf("error opening osz (zip) id:%d err: %v", set.ID, err)
	}

	// Create a map to hold the .osu files
	osuFiles := make(map[string][]byte)

	// Iterate over each file in the zip archive
	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, ".osu") {
			continue
		}
		if file.FileInfo().IsDir() {
			Fail("_broken_files", set.ID, file.Name)
			continue
		}
		if strings.Contains(file.Name, "/") || strings.Contains(file.Name, "\\") {
			Fail("_broken_files", set.ID, file.Name)
			continue
		}
		// Open the .osu file inside the zip archive
		fileReader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("error opening .osu file %s: %v", file.Name, err)
		}
		defer fileReader.Close()

		// Read the contents of the .osu file
		fileContents, err := io.ReadAll(fileReader)
		if err != nil {
			return nil, fmt.Errorf("error reading .osu file %s: %v", file.Name, err)
		}

		// Store the file contents in the map
		osuFiles[file.Name] = fileContents
	}

	if len(osuFiles) == 0 {
		PanicF("no .osu files found in the beatmap")
	}

	// Return the map of .osu files
	return osuFiles, nil
}

func DownloadBeatmapsetBytes(beatmapsetId int) ([]byte, error) {
	// Define the URL for the download
	url := fmt.Sprintf("https://osu.ppy.sh/beatmapsets/%d/download", beatmapsetId)

	Throttle()
	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set the necessary headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,lt-LT;q=0.8,lt;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("DNT", "1")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Referer", fmt.Sprintf("https://osu.ppy.sh/beatmapsets/%d", beatmapsetId))
	req.Header.Set("Sec-CH-UA", `"Not;A=Brand";v="99", "Google Chrome";v="139", "Chromium";v="139"`)
	req.Header.Set("Sec-CH-UA-Arch", `"arm"`)
	req.Header.Set("Sec-CH-UA-Bitness", `"64"`)
	req.Header.Set("Sec-CH-UA-Full-Version", `"139.0.7258.155"`)
	req.Header.Set("Sec-CH-UA-Full-Version-List", `"Not;A=Brand";v="99.0.0.0", "Google Chrome";v="139.0.7258.155", "Chromium";v="139.0.7258.155"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Model", `""`)
	req.Header.Set("Sec-CH-UA-Platform", `"macOS"`)
	req.Header.Set("Sec-CH-UA-Platform-Version", `"15.6.1"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	// Set the cookies (replacing the value with the correct session cookie)
	req.AddCookie(&http.Cookie{
		Name:  "osu_session",
		Value: OsuSession,
	})

	// Perform the request
	client := &http.Client{Timeout: time.Minute * 10}
	fmt.Printf("downloading set %d\n", beatmapsetId)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Return the downloaded data as a byte slice
	return body, nil
}
