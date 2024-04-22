package util

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/log"
)

// Command defines a command that should be executed.
type Command struct {
	// Command holds the command with its optional arguments.
	Command []string

	// Directory defines the directory the execution should run in, without changing the working directory of the caller.
	Directory string
}

// CommandWithResult executes a command and returns its output, while printing the same output to the given logger.
func CommandWithResult(logger *log.Logger, command *Command) (stdout string, stderr string, err error) {
	logger.Printf("$ %s", strings.Join(command.Command, " "))

	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	c := exec.Command(command.Command[0], command.Command[1:]...)
	if command.Directory != "" {
		c.Dir = command.Directory
	}
	c.Stdout = io.MultiWriter(logger.Writer(), &stdoutWriter)
	c.Stderr = io.MultiWriter(logger.Writer(), &stderrWriter)

	if err := c.Run(); err != nil {
		return stdoutWriter.String(), stderrWriter.String(), pkgerrors.WithStack(err)
	}

	return stdoutWriter.String(), stderrWriter.String(), nil
}
