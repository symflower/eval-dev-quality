package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/util"
)

var whitelist = []string{
	// Testing, debugging and logging is fine.
	"/testing/",
	"log/",
	"util/debug",
	// Functions that serve as potential APIs for the future are fine.
	`provider/symflower/symflower\.go.*NewProvider`,
	`model/symflower/symflower\.go.*NewModelWithTimeout`,
	`tools/install\.go.*Install`,
}

func main() {
	logBuffer, logger := log.Buffer()
	output, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"deadcode",
			"./...",
		},
	})
	if err != nil {
		panic([]any{err, logBuffer.String()})
	}

	var matches []string
MATCHES:
	for _, deadcode := range strings.Split(output, "\n") {
		if strings.TrimSpace(deadcode) == "" {
			continue
		}

		for _, filter := range whitelist {
			if regexp.MustCompile(filter).MatchString(deadcode) {
				continue MATCHES
			}
		}

		matches = append(matches, deadcode)
	}

	for _, match := range matches {
		fmt.Println(match)
	}
	if len(matches) > 0 {
		os.Exit(-1)
	}
}
