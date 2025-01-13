package cmd

import (
	"github.com/symflower/eval-dev-quality/evaluate"
	"github.com/symflower/eval-dev-quality/log"
)

// Version defines the "version" command.
type Version struct {
	// logger holds the logger of the command.
	logger *log.Logger
}

var _ SetLogger = (*Version)(nil)

// SetLogger sets the logger of the command.
func (command *Version) SetLogger(logger *log.Logger) {
	command.logger = logger
}

// Execute executes the command.
func (command *Version) Execute(args []string) (err error) {
	revision := evaluate.Revision
	if revision == "" {
		revision = "development"
	}

	command.logger.PrintfWithoutMeta("eval-dev-quality version %s - revision %s\n", evaluate.Version, revision)

	return nil
}
