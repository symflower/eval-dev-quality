package tools

import (
	"github.com/symflower/eval-dev-quality/log"
)

// Tool defines an external tool.
type Tool interface {
	// BinaryName returns the name of the tool's binary.
	BinaryName() string
	// BinaryPath returns the file path of the tool's binary or the command name that should be executed.
	// The binary path might also be just the binary name in case the tool is expected to be on the system path.
	BinaryPath() string

	// CheckVersion checks if the tool's version is compatible with the required version.
	CheckVersion(logger *log.Logger, binaryPath string) error
	// RequiredVersion returns the required version of the tool.
	RequiredVersion() string

	// Install installs the tool's binary to the given install path.
	Install(logger *log.Logger, installPath string) error
}
