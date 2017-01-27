package reddit

import (
	"strings"
)

// PostInfo represents info about a post that a user has voted on
type PostInfo struct {
	Username string
	Vote     string
	Title    string
	Content  string
}

// GetPostInfo processes a line in the csv and returns a PostInfo struct
func GetPostInfo(input string) PostInfo {
	data := strings.Split(input, ",")
	return PostInfo{data[0], data[2], data[1][3:], "No content"}
}
