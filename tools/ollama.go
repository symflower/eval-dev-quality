package tools

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"runtime"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"golang.org/x/mod/semver"

	"github.com/symflower/eval-dev-quality/util"
)

// ollama holds the "Ollama" tool.
type ollama struct{}

func init() {
	Register(NewOllama())
}

// NewOllama returns a new Ollama tool.
func NewOllama() Tool {
	return &ollama{}
}

var _ Tool = &ollama{}

// BinaryName returns the name of the tool's binary.
func (*ollama) BinaryName() string {
	return "ollama" + osutil.BinaryExtension()
}

// OllamaPath holds the file path to the Ollama binary or the command name that should be executed.
var OllamaPath = "ollama" + osutil.BinaryExtension()

// BinaryPath returns the file path of the tool's binary or the command name that should be executed.
// The binary path might also be just the binary name in case the tool is expected to be on the system path.
func (*ollama) BinaryPath() string {
	return OllamaPath
}

// CheckVersion checks if the tool's version is compatible with the required version.
func (*ollama) CheckVersion(logger *log.Logger, binaryPath string) (err error) {
	output, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			binaryPath,
			"--version",
		},
	})
	if err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, `unable to verify "Ollama" installation`))
	}

	m := regexp.MustCompile(`version is (\d+\.\d+\.\d+)`).FindStringSubmatch(output)
	if m == nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(errors.New("cannot find version"), output))
	}

	versionPresent := "v" + m[1]
	if !semver.IsValid(versionPresent) {
		pkgerrors.New(fmt.Sprintf("invalid semantic version string %q", versionPresent))
	}
	versionRequired := "v" + ollamaVersion
	if !semver.IsValid(versionRequired) {
		pkgerrors.New(fmt.Sprintf("invalid semantic version string %q", versionRequired))
	}

	// Check if binary is compatible.
	if semver.Compare(versionPresent, versionRequired) < 0 {
		return pkgerrors.WithStack(ErrToolVersionOutdated)
	}

	return nil
}

// ollamaVersion holds the version of Ollama required for this revision of the evaluation.
var ollamaVersion = "0.1.38"

// RequiredVersion returns the required version of the tool.
func (*ollama) RequiredVersion() string {
	return ollamaVersion
}

// Install installs the tool's binary to the given install path.
func (*ollama) Install(logger *log.Logger, installPath string) (err error) {
	if !osutil.IsLinux() {
		return pkgerrors.WithMessage(pkgerrors.WithStack(ErrUnsupportedOperatingSystem), runtime.GOOS)
	}

	var architectureIdentifier string
	switch a := runtime.GOARCH; a {
	case "amd64":
		architectureIdentifier = "amd64"
	case "arm64":
		architectureIdentifier = "arm64"
	default:
		return pkgerrors.WithStack(fmt.Errorf("unsupported architecture %s", a))
	}

	ollamaDownloadURL := "https://github.com/ollama/ollama/releases/download/v" + ollamaVersion + "/ollama-linux-" + architectureIdentifier
	ollamaInstallPath := filepath.Join(installPath, "ollama")
	logger.Printf("Install \"ollama\" to %s from %s", ollamaInstallPath, ollamaDownloadURL)
	if err := osutil.DownloadFileWithProgress(ollamaDownloadURL, ollamaInstallPath); err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot download to %s from %s", ollamaInstallPath, ollamaDownloadURL)))
	}

	// Non-Windows binaries need to be made executable because the executable bit is not set for downloads.
	if !osutil.IsWindows() {
		if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
			Command: []string{"chmod", "+x", ollamaInstallPath},
		}); err != nil {
			return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot make %s executable", ollamaInstallPath)))
		}
	}

	return nil
}
