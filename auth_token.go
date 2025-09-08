package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	AuthToken = fetchToken(context.Background(), ClientID, ClientSecret)
)

// TokenResponse models the osu! OAuth token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

func fetchToken(ctx context.Context, clientId int, clientSecret string) *TokenResponse {
	form := url.Values{}
	form.Set("client_id", strconv.Itoa(clientId))
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "public")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://osu.ppy.sh/oauth/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		panic(fmt.Errorf("build request: %w", err))
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// You can pass your own http.Client if you prefer.
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("send request: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		panic(fmt.Errorf("osu oauth error: status %d, body: %s", resp.StatusCode, string(body)))
	}

	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		panic(fmt.Errorf("decode token: %w", err))
	}
	return &tok
}
