package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

var reTaskFileName = regexp.MustCompile(`Given the following \w+ code file "(.*)" with`)

func parseTaskFileName(s string) (string, bool) {
	if match := reTaskFileName.FindStringSubmatch(s); match != nil {
		return match[1], true
	}

	return "", false
}

var reCoverageObjects = regexp.MustCompile(`Executes tests with (\d+) coverage objects`)

func parseCoverageObjects(s string) (int, bool) {
	match := reCoverageObjects.FindStringSubmatch(s)
	if match == nil {
		return 0, false
	}
	coverageObjectsText := match[1]
	coverageObjects, err := strconv.Atoi(coverageObjectsText)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %q to integer (%q)", coverageObjectsText, s))
	}

	return coverageObjects, true
}

func main() {
	logFileNames := os.Args[1:]
	var logFileData []byte

	for _, logFileName := range logFileNames {
		data, err := os.ReadFile(logFileName)
		if err != nil {
			panic(err)
		}
		logFileData = append(logFileData, data...)
	}

	var currentTaskFileName string
	bestCoverage := map[string]int{}

	logFileLines := strings.Split(string(logFileData), "\n")
	fmt.Printf("Loaded %d log files (total of %d lines)\n", len(logFileNames), len(logFileLines))
	for _, line := range logFileLines {
		if taskFileName, ok := parseTaskFileName(line); ok {
			currentTaskFileName = taskFileName
		}

		if currentCoverageObjects, ok := parseCoverageObjects(line); ok && currentCoverageObjects > bestCoverage[currentTaskFileName] {
			bestCoverage[currentTaskFileName] = currentCoverageObjects
		}
	}

	sum := 0
	taskFileNames := maps.Keys(bestCoverage)
	sort.Strings(taskFileNames)
	for _, taskFileName := range taskFileNames {
		score := bestCoverage[taskFileName]
		fmt.Printf(" - %q:%d\n", taskFileName, score)
		sum += score
	}
	fmt.Printf("∑ = %d (weighted: %d)\n", sum, sum*10)
}
