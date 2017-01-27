package reddit

import (
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

var (
	rateUsed                    = 0
	rateRemaining               = 60
	rateReset     time.Duration = 60
)

// GetPostInfo processes a line in the csv and returns a PostInfo struct
func GetPostInfo(input string) PostInfo {
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
		title, content = getRedditInfo(fullname)
	} else {
		// Wait until new period
		time.Sleep(rateReset * time.Second)
		// Make request
		title, content = getRedditInfo(fullname)
	}
	response.Title = title
	response.Content = content
	return *response
}

func getRedditInfo(fullname string) (title, content string) {
	return
}
