package reddit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PostInfo represents info about a post that a user has voted on
type PostInfo struct {
	Username string
	Vote     string
	Title    string
	Content  string
}

// APIConfig declares a configuration nessessary to make API calls to Reddit
type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"access_token"`
	ID       string `yaml:"client_id"`
	Secret   string `yaml:"client_secret"`
}

const (
	tokenURL      = "https://www.reddit.com/api/v1/access_token"
	lookupURL     = "https://oauth.reddit.com/by_id/"
	userAgent     = "RNNScraperBot/0.1 by "
	authHeaderVal = "bearer "

	headerUsed = "X-Ratelimit-Used"
	headerRem  = "X-Ratelimit-Remaining"
	headerNext = "X-Ratelimit-Reset"
)

var (
	rateUsed      = 0
	rateRemaining = 60
	rateReset     = 60

	accessToken     string
	tokenExpiration time.Time
)

// GetPostInfo processes a line in the csv and returns a PostInfo struct
func GetPostInfo(input string, config APIConfig) PostInfo {
	response := new(PostInfo)

	// Process input
	data := strings.Split(input, ",")
	username := data[0]
	vote := data[2]
	fullname := data[1]
	response.Username = username
	response.Vote = vote

	var title, content string

	// Get data from Reddit
	if rateRemaining > 0 {
		// Make request
		updateAccessToken(config)
		title, content = getRedditInfo(fullname, config)
	} else {
		fmt.Printf("Rate exceeded, waiting %d seconds.\n", rateReset)
		// Wait until new period
		time.Sleep(time.Duration(rateReset) * time.Second)
		// Make request
		updateAccessToken(config)
		title, content = getRedditInfo(fullname, config)
	}
	response.Title = title
	response.Content = content
	return *response
}

// Updates the access token if it's invalid
func updateAccessToken(config APIConfig) {
	if accessToken == "" || time.Now().After(tokenExpiration) {
		// TODO Update token and expiration time
		client := &http.Client{}
		data := url.Values{}
		data.Set("grant_type", "password")
		data.Set("username", config.Username)
		data.Set("password", config.Password)
		req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
		if err != nil {
			panic(err)
		}
		req.Header.Set("User-Agent", userAgent+config.Username)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		req.SetBasicAuth(config.ID, config.Secret)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		// Too many requests
		if resp.StatusCode == 429 {
			time.Sleep(500 * time.Millisecond)
			updateAccessToken(config)
			return
		}
		if resp.StatusCode != 200 {
			panic(resp.Status)
		}
		var accessResponse TokenResponse
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &accessResponse)
		accessToken = accessResponse.AccessToken
		// Set new expiration time, minus 1 minute for safety
		tokenExpiration = time.Now().Add(time.Duration(accessResponse.ExpiresIn - 60))
	}
}

func getRedditInfo(fullname string, config APIConfig) (title, content string) {
	client := http.Client{}
	req, err := http.NewRequest("GET", lookupURL+fullname, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", userAgent+config.Username)
	req.Header.Set("Authorization", authHeaderVal+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode == 401 {
		updateAccessToken(config)
		return getRedditInfo(fullname, config)
	}
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}
	defer resp.Body.Close()
	rateRemaining, _ = strconv.Atoi(resp.Header.Get(headerRem))
	rateUsed, _ = strconv.Atoi(resp.Header.Get(headerUsed))
	rateReset, _ = strconv.Atoi(resp.Header.Get(headerNext))
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	// TODO: Process body to extract data
	return
}
