package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"

	"github.com/kandidat-highlights/reddit-scraper/reddit"

	"flag"

	"os"
	"os/signal"

	"encoding/csv"

	"time"

	"strings"

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
	batchSize     int
	currentPos    int64
)

func main() {
	// Parse input flags
	flag.Int64Var(&startPos, "start", 0, "Line to start processing at")
	flag.IntVar(&batchSize, "batch", 25, "Determines how large batches will be processed")
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
	log.Println("Completed file processing")
	log.Println("==========================")
	log.Printf("Stopped at position: %d\n", currentPos)
}

func processRaw(rawLine string) (username, vote, fullname string) {
	line := strings.Split(rawLine, ",")
	username = line[0]
	fullname = line[1]
	vote = line[2]
	return
}

func processBatch(batch reddit.InputBatch, outputWriter *csv.Writer) {
	responses, err := reddit.GetPostInfo(batch, config)
	if err == nil {
		for _, post := range responses {
			outputWriter.Write([]string{post.Username, post.Vote, post.SubReddit, post.Title, post.Content})
		}
		outputWriter.Flush()
	}
}

func process(input io.ReadSeeker, outputFile *os.File, start int64) error {
	// Start looking at position 'start' in the file
	if _, err := input.Seek(start, 0); err != nil {
		return err
	}

	// Initialize vars
	scanner := bufio.NewScanner(input)
	currentPos = start
	outputWriter := csv.NewWriter(outputFile)
	defer outputWriter.Flush()

	// Setup scanner to keep track of where it is
	scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		currentPos += int64(advance)
		return
	}
	scanner.Split(scanLines)

	// Start processing file
	batch := new(reddit.InputBatch)
	for shouldProcess && scanner.Scan() {
		log.Printf("On position: %d\n", currentPos)
		u, v, id := processRaw(scanner.Text())
		inData := reddit.Input{Username: u, Vote: v, FullName: id}
		*batch = append(*batch, inData)

		if len(*batch) >= batchSize {
			processBatch(*batch, outputWriter)
			batch = new(reddit.InputBatch)
			// Wait so API is not overloaded
			time.Sleep(1 * time.Second)
		}
	}
	// Process any remaining posts in batch
	if len(*batch) > 0 {
		processBatch(*batch, outputWriter)
		time.Sleep(1 * time.Second)
	}
	onDone()
	return scanner.Err()
}
