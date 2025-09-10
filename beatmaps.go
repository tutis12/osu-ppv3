package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Beatmap represents the detailed structure of a beatmap in the osu! API.
type Beatmap struct {
	Accuracy             float64    `json:"accuracy"`
	Ar                   float64    `json:"ar"`
	Beatmapset           Beatmapset `json:"beatmapset"`
	BeatmapsetID         int        `json:"beatmapset_id"`
	Bpm                  float64    `json:"bpm"`
	Checksum             string     `json:"checksum"`
	Convert              bool       `json:"convert"`
	CountCircles         int        `json:"count_circles"`
	CountSliders         int        `json:"count_sliders"`
	CountSpinners        int        `json:"count_spinners"`
	Cs                   float64    `json:"cs"`
	CurrentUserPlaycount int        `json:"current_user_playcount"`
	DeletedAt            *time.Time `json:"deleted_at"`
	DifficultyRating     float64    `json:"difficulty_rating"`
	Drain                float64    `json:"drain"`
	Failtimes            Failtimes  `json:"failtimes"`
	HitLength            int        `json:"hit_length"`
	ID                   int        `json:"id"`
	IsScoreable          bool       `json:"is_scoreable"`
	LastUpdated          time.Time  `json:"last_updated"`
	MaxCombo             int        `json:"max_combo"`
	Mode                 string     `json:"mode"`
	ModeInt              int        `json:"mode_int"`
	Owners               []Owner    `json:"owners"`
	Passcount            int        `json:"passcount"`
	Playcount            int        `json:"playcount"`
	Ranked               int        `json:"ranked"`
	Status               string     `json:"status"`
	TotalLength          int        `json:"total_length"`
	URL                  string     `json:"url"`
	UserID               int        `json:"user_id"`
	Version              string     `json:"version"`
}

// Beatmapset represents detailed information about a beatmapset.
type Beatmapset struct {
	Artist             string             `json:"artist"`
	ArtistUnicode      string             `json:"artist_unicode"`
	Availability       Availability       `json:"availability"`
	Bpm                float64            `json:"bpm"`
	CanBeHyped         bool               `json:"can_be_hyped"`
	Covers             Covers             `json:"covers"`
	Creator            string             `json:"creator"`
	DeletedAt          *time.Time         `json:"deleted_at"`
	DiscussionEnabled  bool               `json:"discussion_enabled"`
	DiscussionLocked   bool               `json:"discussion_locked"`
	FavouriteCount     int                `json:"favourite_count"`
	GenreID            int                `json:"genre_id"`
	Hype               *HypeCounter       `json:"hype"`
	ID                 int                `json:"id"`
	IsScoreable        bool               `json:"is_scoreable"`
	LanguageID         int                `json:"language_id"`
	LastUpdated        time.Time          `json:"last_updated"`
	LegacyThreadURL    string             `json:"legacy_thread_url"`
	NominationsSummary NominationsSummary `json:"nominations_summary"`
	NSFW               bool               `json:"nsfw"`
	Offset             int                `json:"offset"`
	PlayCount          int                `json:"play_count"`
	PreviewURL         string             `json:"preview_url"`
	Ranked             int                `json:"ranked"`
	RankedDate         time.Time          `json:"ranked_date"`
	Rating             float64            `json:"rating"`
	Ratings            []int              `json:"ratings"`
	Source             string             `json:"source"`
	Spotlight          bool               `json:"spotlight"`
	Status             string             `json:"status"`
	Storyboard         bool               `json:"storyboard"`
	SubmittedDate      time.Time          `json:"submitted_date"`
	Tags               string             `json:"tags"`
	Title              string             `json:"title"`
	TitleUnicode       string             `json:"title_unicode"`
	TrackID            *int               `json:"track_id"`
	UserID             int                `json:"user_id"`
	Video              bool               `json:"video"`
}

type HypeCounter struct {
	Current  int `json:"current"`
	Required int `json:"required"`
}

// Availability represents availability settings of a beatmap.
type Availability struct {
	DownloadDisabled bool    `json:"download_disabled"`
	MoreInformation  *string `json:"more_information"`
}

// Covers represents the different cover images associated with a beatmapset.
type Covers struct {
	Card        string `json:"card"`
	Card2x      string `json:"card@2x"`
	Cover       string `json:"cover"`
	Cover2x     string `json:"cover@2x"`
	List        string `json:"list"`
	List2x      string `json:"list@2x"`
	Slimcover   string `json:"slimcover"`
	Slimcover2x string `json:"slimcover@2x"`
}

// NominationsSummary represents the nomination summary for a beatmapset.
type NominationsSummary struct {
	Current              int          `json:"current"`
	EligibleMainRulesets []string     `json:"eligible_main_rulesets"`
	RequiredMeta         RequiredMeta `json:"required_meta"`
}

// RequiredMeta represents the meta data required for a beatmapset to be ranked.
type RequiredMeta struct {
	MainRuleset    int `json:"main_ruleset"`
	NonMainRuleset int `json:"non_main_ruleset"`
}

// Failtimes represents the failure times for a beatmap.
type Failtimes struct {
	Exit []int `json:"exit"`
	Fail []int `json:"fail"`
}

// Owner represents the owner information of a beatmap.
type Owner struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func FetchBeatmaps(ctx context.Context, ids []int) []Beatmap {
	if len(ids) == 0 {
		panic("")
	}
	start := time.Now()
	var err error
	for {
		var beatmaps []Beatmap
		beatmaps, err = tryFetchBeatmaps(ctx, ids)
		if err == nil {
			if len(beatmaps) == 0 {
				panic("")
			}
			return beatmaps
		}
		time.Sleep(max(time.Second, min(time.Minute, time.Since(start))))
		if strings.Contains(err.Error(), "Too Many Attempts") ||
			strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "Cloudflare encountered an error processing this request") {
			fmt.Println(err.Error())
			os.Exit(2)
			continue
		}
		panic(err)
	}
}

// FetchBeatmaps retrieves beatmap data for the specified beatmap IDs.
func tryFetchBeatmaps(ctx context.Context, ids []int) ([]Beatmap, error) {
	// Ensure the slice has at most 50 IDs
	if len(ids) > 50 {
		panic(fmt.Errorf("cannot request more than 50 beatmaps at once"))
	}

	// Prepare the query parameters
	params := url.Values{}
	for _, id := range ids {
		params.Add("ids[]", fmt.Sprintf("%d", id))
	}

	// Construct the request URL
	url := fmt.Sprintf("https://osu.ppy.sh/api/v2/beatmaps?%s", params.Encode())

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", AuthToken.TokenType+" "+AuthToken.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: time.Minute * 2}

	done := GetToken()
	Throttle()
	defer done()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %s", body)
	}

	type beatmapsStruct struct {
		Beatmaps []Beatmap `json:"beatmaps"`
	}

	fmt.Println("what ", string(body), ids)
	var beatmaps beatmapsStruct
	if err := json.Unmarshal(body, &beatmaps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, body=%s", err, string(body))
	}

	return beatmaps.Beatmaps, nil
}
