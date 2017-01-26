package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"

	"flag"

	"os"
	"os/signal"

	"gopkg.in/yaml.v2"
)

// APIConfig declares a configuration nessessary to make API calls to Reddit
type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"access_token"`
	Secret   string `yaml:"client_secret"`
}

const (
	configPath = "auth.yaml"
	votesPath  = "votes.csv"
	targetPath = "processed.csv"
)

var (
	config     APIConfig
	startLine  int64
	currentPos int64
)

func main() {
	// Parse input flags
	flag.Int64Var(&startLine, "start", 0, "Line to start processing at")
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
	fmt.Println(config)

	// Make sure user knows last processed position if Interupted
	setupInterupCapture()

	// Open the votes for processing
	file, err := os.Open(votesPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Start processing votes at specified position
	err = process(file, startLine)
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
		onInterupt()
		os.Exit(0)
	}()
}

// Function that runs when user interupts program
func onInterupt() {
	fmt.Println("Interupted file processing")
	fmt.Println("==========================")
	fmt.Printf("Stopped at position: %d\n", currentPos)
}

func process(input io.ReadSeeker, start int64) error {
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

	for scanner.Scan() {
		// TODO: Fetch title from Reddit and save to new file
		fmt.Printf("Scanned: %s\n", scanner.Text())
	}
	return scanner.Err()
}
