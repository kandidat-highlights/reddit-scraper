package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hsson/reddit-scraper/reddit"

	"flag"

	"os"
	"os/signal"

	"encoding/csv"

	"time"

	"gopkg.in/yaml.v2"
)

const (
	configPath = "auth.yaml"
	votesPath  = "votes.csv"
	targetPath = "processed.csv"
)

var (
	shouldProcess = true
	config        reddit.APIConfig
	startPos      int64
	currentPos    int64
)

func main() {
	// Parse input flags
	flag.Int64Var(&startPos, "start", 0, "Line to start processing at")
	flag.Parse()

	// Read config from file
	rawConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		panic(err)
	}

	// Make sure user knows last processed position if Interupted
	setupInterupCapture()

	// Open the votes for processing
	file, err := os.Open(votesPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Initialize output file
	outputFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	// Start processing votes at specified position
	err = process(file, outputFile, startPos)
	if err != nil {
		panic(err)
	}
}

// Hook the os interupt signal and process it
func setupInterupCapture() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		shouldProcess = false
	}()
}

// Function that runs when user interupts program
func onDone() {
	fmt.Println("Completed file processing")
	fmt.Println("==========================")
	fmt.Printf("Stopped at position: %d\n", currentPos)
}

func process(input io.ReadSeeker, outputFile *os.File, start int64) error {
	if _, err := input.Seek(start, 0); err != nil {
		return err
	}
	scanner := bufio.NewScanner(input)

	currentPos = start
	scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		currentPos += int64(advance)
		return
	}
	scanner.Split(scanLines)
	outputWriter := csv.NewWriter(outputFile)
	defer outputWriter.Flush()
	for shouldProcess && scanner.Scan() {
		fmt.Printf("On position: %d\n", currentPos)
		info, err := reddit.GetPostInfo(scanner.Text(), config)
		if err == nil {
			outputWriter.Write([]string{info.Username, info.Vote, info.SubReddit, info.Title, info.Content})
			outputWriter.Flush()
		}
		time.Sleep(1 * time.Second)
	}
	onDone()
	return scanner.Err()
}
