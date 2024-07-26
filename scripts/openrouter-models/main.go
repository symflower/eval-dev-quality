package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider/openrouter"
)

var ignoredModels = []string{
	// Alias models.
	"openrouter/openrouter/flavor-of-the-week",
	"openrouter/openrouter/auto",

	// Special property models.
	".*:free$",
	".*:beta$",
	".*:nitro$",
	".*:extended$",

	// Model previews, online access models and vision models.
	".*-preview",
	".*-online",
	".*-vision-?",
}

func isIgnored(model model.Model) bool {
	for _, regex := range ignoredModels {
		if regexp.MustCompile(regex).MatchString(model.ID()) {
			return true
		}
	}

	return false
}

func main() {
	provider := openrouter.NewProvider()
	allModels, err := provider.Models()
	if err != nil {
		panic(err)
	}

	var models []model.Model
	for _, model := range allModels {
		if isIgnored(model) {
			fmt.Printf("ignoring %q\n", model.ID())

			continue
		}

		models = append(models, model)
	}
	slices.SortFunc(models, func(a, b model.Model) int {
		return strings.Compare(a.ID(), b.ID())
	})

	csvFile, err := os.Create("openrouter.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	csvWriter.Write([]string{"model"})
	for _, model := range models {
		csvWriter.Write([]string{model.ID()})
	}
}
