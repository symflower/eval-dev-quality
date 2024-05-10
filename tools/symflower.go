package tools

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/lockfile"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/util"
)

// SymflowerPath holds the file path to the Symflower binary or the command name that should be executed.
var SymflowerPath = "symflower" + osutil.BinaryExtension()

// SymflowerVersion holds the version of Symflower required for this revision of the evaluation.
const SymflowerVersion = "36486"

// SymflowerInstall checks if the "symflower" binary has been installed, and if yes, updates it if necessary and possible.
func SymflowerInstall(logger *log.Logger, installPath string) (err error) {
	// If the Symflower binary is overwritten, make sure it is a file path.
	if SymflowerPath != "symflower"+osutil.BinaryExtension() {
		if osutil.FileExists(SymflowerPath) != nil {
			return pkgerrors.WithStack(pkgerrors.WithMessage(err, "Symflower binary is not a valid file path"))
		}

		logger.Printf("Using Symflower binary %s", SymflowerPath)

		return nil
	}

	installPath, err = filepath.Abs(installPath)
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	if err := os.MkdirAll(installPath, 0755); err != nil {
		return pkgerrors.WithStack(err)
	}

	// Make sure only one process is installing a tool at the same time.
	lock, err := lockfile.New(filepath.Join(installPath, "install.lock"))
	if err != nil {
		return pkgerrors.WithStack(err)
	}
	for {
		if err := lock.TryLock(); err == nil {
			break
		}

		logger.Printf("Try to lock %s for installing but need to wait for another process", installPath)
		time.Sleep(time.Second)
	}
	defer func() {
		if e := lock.Unlock(); e != nil {
			err = errors.Join(err, e)
		}
	}()

	// Check if install path is already used for binaries, or add it if not.
	installPathUsed := false
	for _, p := range strings.Split(os.Getenv(osutil.EnvironmentPathIdentifier), string(os.PathListSeparator)) {
		p = filepath.Clean(p)
		if p == installPath {
			installPathUsed = true

			break
		}
	}
	if !installPathUsed {
		os.Setenv(osutil.EnvironmentPathIdentifier, strings.Join([]string{os.Getenv(osutil.EnvironmentPathIdentifier), installPath}, string(os.PathListSeparator))) // Add the install path last, so we are not overwriting other binaries.
	}

	// Check if the "symflower" binary can already be used.
	symflowerPath, err := exec.LookPath("symflower" + osutil.BinaryExtension())
	if err == nil {
		logger.Printf("Checking \"symflower\" binary %s", symflowerPath)

		symflowerVersionOutput, err := util.CommandWithResult(logger, &util.Command{
			Command: []string{symflowerPath, "version"},
		})
		if err != nil {
			return pkgerrors.WithStack(err)
		}

		// Development version of Symflower is always OK to use.
		if strings.Contains(symflowerVersionOutput, " development on") {
			if !strings.Contains(symflowerVersionOutput, "symflower-local development on") {
				return pkgerrors.WithStack(errors.New("allow Symflower binary to be used concurrently with its shared folder"))
			}

			return nil
		}

		m := regexp.MustCompile(`symflower v(\d+) on`).FindStringSubmatch(symflowerVersionOutput)
		if m == nil {
			return pkgerrors.WithStack(pkgerrors.WithMessage(errors.New("cannot find version"), symflowerVersionOutput))
		}

		// Currently the Symflower version is only one integer, so do a poor-man's version comparision.
		symflowerVersionInstalled, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			return pkgerrors.WithStack(err)
		}
		symflowerVersionWanted, err := strconv.ParseUint(SymflowerVersion, 10, 64)
		if err != nil {
			return pkgerrors.WithStack(err)
		}

		// Binary is installed in a compatible version.
		if symflowerVersionInstalled >= symflowerVersionWanted {
			return nil
		}

		// If the binary got installed by the user, let the user handle the update.
		if filepath.Dir(symflowerPath) != installPath {
			return pkgerrors.WithStack(fmt.Errorf("found \"symflower\" binary with version %d but need at least %d", symflowerVersionInstalled, symflowerVersionWanted))
		}
	}

	// Install Symflower, as it is either outdated or not installed at all.
	symflowerInstallPath := filepath.Join(installPath, "symflower"+osutil.BinaryExtension())
	osIdentifier := runtime.GOOS
	var architectureIdentifier string
	switch a := runtime.GOARCH; a {
	case "386":
		architectureIdentifier = "x86"
	case "amd64":
		architectureIdentifier = "x86_64"
	case "arm":
		architectureIdentifier = "arm"
	case "arm64":
		architectureIdentifier = "arm64"
	default:
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("unkown architecture %s", a)))
	}

	symflowerDownloadURL := "https://download.symflower.com/local/v" + SymflowerVersion + "/symflower-" + osIdentifier + "-" + architectureIdentifier + osutil.BinaryExtension()
	logger.Printf("Install \"symflower\" to %s from %s", symflowerInstallPath, symflowerDownloadURL)
	if err := osutil.DownloadFileWithProgress(symflowerDownloadURL, symflowerInstallPath); err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot download to %s from %s", symflowerInstallPath, symflowerDownloadURL)))
	}

	// Non-Windows binaries need to be made executable because the executable bit is not set for downloads.
	if !osutil.IsWindows() {
		if _, err := util.CommandWithResult(logger, &util.Command{
			Command: []string{"chmod", "+x", symflowerInstallPath},
		}); err != nil {
			return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot make %s executable", symflowerInstallPath)))
		}
	}

	return nil
}
