package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// JSONModels holds the collection of the models.
type JSONModels struct {
	Models []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"models"`
}

// FullOllamaModels holds a full Ollama model collection.
type FullOllamaModels struct {
	Models []Model `json:"models"`
}

// Model holds a model.
type Model struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        []Tag  `json:"tags"`
}

// Tag holds a tag.
type Tag struct {
	Name string  `json:"name"`
	Size float64 `json:"size"`
}

// Fetches all the ollama models and then scrapes their detail page to get the information about the sizes. Stores the result in a CSV file.
func main() {
	// Fetch a list of all models.
	ollamaModels := JSONModels{}
	modelsBody := OnPage("https://ollama-models.zwz.workers.dev/")
	if err := json.Unmarshal([]byte(modelsBody), &ollamaModels); err != nil {
		log.Fatal(err)
	}

	// Fetch details for each model
	fullModels := FullOllamaModels{}
	for _, model := range ollamaModels.Models {
		newModel := Model{
			Name:        model.Name,
			Description: model.Description,
		}

		tagsBody := OnPage("https://ollama.com/library/" + model.Name + "/tags")

		split := strings.Split(stripHTMLRegex(tagsBody), " ")

		for _, tag := range model.Tags {
			textSize := ""
			for i, l := range split {
				if l == tag {
					textSize = split[i+3]
					if textSize == "•" {
						textSize = split[i+4]
					}
				}
			}

			// Size to MB
			var size float64
			if strings.HasSuffix(textSize, "GB") {
				textSize = strings.TrimSuffix(textSize, "GB")
				parsed, err := strconv.ParseFloat(textSize, 64)
				if err != nil {
					log.Fatal(err)
				}
				size = parsed
			}
			if strings.HasSuffix(textSize, "MB") {
				textSize = strings.TrimSuffix(textSize, "MB")
				parsed, err := strconv.ParseFloat(textSize, 64)
				if err != nil {
					log.Fatal(err)
				}
				size = parsed / 1024
			}

			newModel.Tags = append(newModel.Tags, Tag{
				Name: tag,
				Size: size,
			})
		}

		fullModels.Models = append(fullModels.Models, newModel)
	}

	file, err := os.Create("model-size.csv")
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	csvwriter := csv.NewWriter(file)

	if err := csvwriter.Write([]string{"model", "size (GB)"}); err != nil {
		log.Fatal(err)
	}
	for _, m := range fullModels.Models {
		for _, t := range m.Tags {
			if err := csvwriter.Write([]string{m.Name + "/" + t.Name, fmt.Sprintf("%.2f", t.Size)}); err != nil {
				log.Fatal(err)
			}
		}
	}
	csvwriter.Flush()
}

// OnPage returns the body of an URL.
func OnPage(link string) string {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	content, err := io.ReadAll(res.Body)
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func stripHTMLRegex(input string) string {
	input = regexp.MustCompile(`<.*?>`).ReplaceAllString(input, "")
	input = strings.ReplaceAll(input, ">", "")
	input = strings.ReplaceAll(input, "<", "")

	space := regexp.MustCompile(`\s+`)
	input = space.ReplaceAllString(input, " ")

	return input
}
