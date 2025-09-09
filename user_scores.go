package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetBestScores(userId int) ([]Score, error) {
	scores0, err := getBestScoresPage(userId, 100, 0)
	if err != nil {
		return nil, err
	}
	scores1, err := getBestScoresPage(userId, 100, 100)
	if err != nil {
		return nil, err
	}
	return append(scores0, scores1...), nil
}
func getBestScoresPage(userId int, limit int, offset int) ([]Score, error) {
	url := fmt.Sprintf("https://osu.ppy.sh/api/v2/users/%d/scores/best?mode=osu&limit=%d&offset=%d", userId, limit, offset)

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", AuthToken.TokenType+" "+AuthToken.AccessToken)

	done := GetToken()
	Throttle()
	defer done()
	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}

	var scores []Score

	err = json.Unmarshal(body, &scores)
	if err != nil {
		return nil, err
	}

	return scores, nil
}

type Score struct {
	Accuracy              float64     `json:"accuracy"`
	BestID                interface{} `json:"best_id"`
	CreatedAt             time.Time   `json:"created_at"`
	ID                    int64       `json:"id"`
	MaxCombo              int         `json:"max_combo"`
	Mode                  string      `json:"mode"`
	ModeInt               int         `json:"mode_int"`
	Mods                  []string    `json:"mods"`
	Passed                bool        `json:"passed"`
	Perfect               bool        `json:"perfect"`
	PP                    float64     `json:"pp"`
	Rank                  string      `json:"rank"`
	Replay                bool        `json:"replay"`
	Score                 int         `json:"score"`
	Statistics            Statistics  `json:"statistics"`
	Type                  string      `json:"type"`
	UserID                int64       `json:"user_id"`
	CurrentUserAttributes struct {
		Pin interface{} `json:"pin"`
	} `json:"current_user_attributes"`
	Beatmap    Beatmap    `json:"beatmap"`
	BeatmapSet BeatmapSet `json:"beatmapset"`
	User       User       `json:"user"`
	Weight     Weight     `json:"weight"`
}

type Statistics struct {
	Count100  int         `json:"count_100"`
	Count300  int         `json:"count_300"`
	Count50   int         `json:"count_50"`
	CountGeki interface{} `json:"count_geki"`
	CountKatu interface{} `json:"count_katu"`
	CountMiss int         `json:"count_miss"`
}

type BeatmapSet struct {
	Artist         string      `json:"artist"`
	ArtistUnicode  string      `json:"artist_unicode"`
	Covers         Covers      `json:"covers"`
	Creator        string      `json:"creator"`
	FavouriteCount int         `json:"favourite_count"`
	GenreID        int         `json:"genre_id"`
	Hype           interface{} `json:"hype"`
	ID             int         `json:"id"`
	LanguageID     int         `json:"language_id"`
	NSFW           bool        `json:"nsfw"`
	Offset         int         `json:"offset"`
	PlayCount      int         `json:"play_count"`
	PreviewURL     string      `json:"preview_url"`
	Source         string      `json:"source"`
	Spotlight      bool        `json:"spotlight"`
	Status         string      `json:"status"`
	Title          string      `json:"title"`
	TitleUnicode   string      `json:"title_unicode"`
	TrackID        interface{} `json:"track_id"`
	UserID         int         `json:"user_id"`
	Video          bool        `json:"video"`
}

type User struct {
	AvatarURL     string      `json:"avatar_url"`
	CountryCode   string      `json:"country_code"`
	DefaultGroup  string      `json:"default_group"`
	ID            int64       `json:"id"`
	IsActive      bool        `json:"is_active"`
	IsBot         bool        `json:"is_bot"`
	IsDeleted     bool        `json:"is_deleted"`
	IsOnline      bool        `json:"is_online"`
	IsSupporter   bool        `json:"is_supporter"`
	LastVisit     time.Time   `json:"last_visit"`
	PMFriendsOnly bool        `json:"pm_friends_only"`
	ProfileColour interface{} `json:"profile_colour"`
	Username      string      `json:"username"`
}

type Weight struct {
	Percentage float64 `json:"percentage"`
	PP         float64 `json:"pp"`
}
