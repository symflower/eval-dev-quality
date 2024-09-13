package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

var excludeTasks = []string{
	"symflower-fix",
	"code-repair",
}
var totalRuns = 5

func main() {
	csvFile, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	records, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		panic(err)
	}
	os.Stderr.WriteString(fmt.Sprintf("Loaded CSV file with %d records\n", len(records)))

	// Collect all results and also the maximum scores.
	scoresPerModelPerRun := map[string][]float64{}
RECORDS:
	for _, record := range records[1:] {
		for _, exclude := range excludeTasks {
			if strings.Contains(record[1]+record[2]+record[3], exclude) {
				continue RECORDS
			}
		}

		model := record[0]
		run, err := strconv.Atoi(record[4])
		if err != nil {
			panic(err)
		}
		run = run - 1
		perRun := scoresPerModelPerRun[model]
		if perRun == nil {
			perRun = make([]float64, totalRuns)
			scoresPerModelPerRun[model] = perRun
		}

		score, err := strconv.ParseInt(record[5], 10, 64)
		if err != nil {
			panic(fmt.Errorf("%s: %q", err, record[5]))
		}
		perRun[run] = perRun[run] + float64(score)
	}

	fmt.Println("model_id,mean,sum,coefficient_of_variation,standard_error,standard_deviation,mean_deviation")

	models := maps.Keys(scoresPerModelPerRun)
	sort.Strings(models)
	for _, model := range models {
		samples := scoresPerModelPerRun[model]
		mean, variance := stat.MeanVariance(samples, nil)
		cov := math.Sqrt(variance) / mean

		sum := floats.Sum(samples)
		stderr := math.Sqrt(float64(len(samples)) * variance)
		var differences float64
		for _, sample := range samples {
			differences += math.Abs(mean - sample)
		}
		meanDeviation := differences / float64(len(samples))

		fmt.Printf("%s,%v,%v,%v,%v,%v,%v\n", model, mean, sum, cov, stderr, math.Sqrt(variance), meanDeviation)
	}
}
