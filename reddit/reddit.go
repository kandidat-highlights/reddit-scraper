package reddit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PostInfo represents info about a post that a user has voted on
type PostInfo struct {
	Username  string
	Vote      string
	SubReddit string
	Title     string
	Content   string
}

// APIConfig declares a configuration nessessary to make API calls to Reddit
type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"access_token"`
	ID       string `yaml:"client_id"`
	Secret   string `yaml:"client_secret"`
}

// Input represents the input from the un-processed data
type Input struct {
	Username string
	Vote     string
	FullName string
}

// InputBatch is a collection of inputs
type InputBatch []Input

const (
	tokenURL      = "https://www.reddit.com/api/v1/access_token"
	lookupURL     = "https://oauth.reddit.com/by_id/"
	userAgent     = "RNNScraperBot/0.1 by "
	authHeaderVal = "bearer "

	headerUsed = "X-Ratelimit-Used"
	headerRem  = "X-Ratelimit-Remaining"
	headerNext = "X-Ratelimit-Reset"

	retryTime  = 5 * time.Second
	retryCount = 5
)

var (
	rateUsed      = 0
	rateRemaining = 60
	rateReset     = 60

	retryAttempts = retryCount

	accessToken     string
	tokenExpiration time.Time
)

// GetPostInfo processes a line in the csv and returns a PostInfo struct
func GetPostInfo(input InputBatch, config APIConfig) ([]PostInfo, error) {

	// Get data from Reddit
	if rateUsed < 60 {
		// Make request
		updateAccessToken(config)
		result, err := getRedditInfo(input, config)
		if err != nil {
			return []PostInfo{}, err
		}
		return result, nil
	}

	fmt.Printf("Rate exceeded, waiting %d seconds.\n", rateReset)
	// Wait until new period
	time.Sleep(time.Duration(rateReset) * time.Second)
	// Make request
	updateAccessToken(config)
	result, err := getRedditInfo(input, config)
	if err != nil {
		return []PostInfo{}, err
	}
	return result, nil
}

// Updates the access token if it's invalid
func updateAccessToken(config APIConfig) {
	if accessToken == "" || time.Now().After(tokenExpiration) {
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

func formatAPIArguments(inputs InputBatch) string {
	fullnames := make([]string, len(inputs))
	for _, in := range inputs {
		fullnames = append(fullnames, in.FullName)
	}
	return strings.Join(fullnames, ",")
}

func getRedditInfo(inputs InputBatch, config APIConfig) ([]PostInfo, error) {
	responses := []PostInfo{}

	// Setup request to API
	client := http.Client{}
	url := lookupURL + formatAPIArguments(inputs)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	// Set requred headers for Reddit API
	req.Header.Set("User-Agent", userAgent+config.Username)
	req.Header.Set("Authorization", authHeaderVal+accessToken)

	// Make the request
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if retryAttempts > 0 {
			retryAttempts--
			log.Printf("An error occured (%d): %v\n", resp.StatusCode, err)
			log.Printf("Trying again in %d seconds\n", retryTime)
			time.Sleep(retryTime)
			updateAccessToken(config)
			return getRedditInfo(inputs, config)
		}
		retryAttempts = retryCount
		log.Println("Can't resolve error, skipping")
		return []PostInfo{}, errors.New("Can't fix error, skipping")
	}

	// Keep track of rate limits
	rateResetTmp, err := strconv.Atoi(resp.Header.Get(headerNext))
	rateResetTmp = int(math.Mod(float64(rateResetTmp), 60.0))
	if rateResetTmp > rateReset && err == nil {
		rateUsed = 0
	}
	rateReset = rateResetTmp
	rateUsed++

	// Start processing response
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var listing RedditListing
	json.Unmarshal(body, &listing)
	if len(listing.Data.Children) == 0 {
		return responses, errors.New("All empty")
	}

	// Create a list of every processed post
	for i, child := range listing.Data.Children {
		postInfo := new(PostInfo)
		postInfo.Username = inputs[i].Username
		postInfo.Vote = inputs[i].Vote
		postInfo.Title = child.Data.Title
		postInfo.SubReddit = child.Data.Subreddit
		postInfo.Content = child.Data.Selftext
		responses = append(responses, *postInfo)
	}

	log.Printf("Processed %d posts, Rate used: %d, Seconds to reset: %d", len(inputs), rateUsed, rateReset)
	return responses, nil
}
