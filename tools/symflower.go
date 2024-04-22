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

// SymflowerVersion holds the version of Symflower required for this revision of the evaluation.
const SymflowerVersion = "35657"

// SymflowerInstall checks if the "symflower" binary has been installed, and if yes, updates it if necessary and possible.
func SymflowerInstall(logger *log.Logger, installPath string) (err error) {
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
	symflowerPath, err := exec.LookPath("symflower")
	if err == nil {
		logger.Printf("Checking \"symflower\" binary %s", symflowerPath)

		symflowerVersionOutput, _, err := util.CommandWithResult(logger, &util.Command{
			Command: []string{symflowerPath, "version"},
		})
		if err != nil {
			return pkgerrors.WithStack(err)
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
	symflowerInstallPath := filepath.Join(installPath, "symflower")
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

	logger.Printf("Install \"symflower\" to %s", symflowerInstallPath)
	if err := osutil.DownloadFileWithProgress("https://download.symflower.com/local/v"+SymflowerVersion+"/symflower-"+osIdentifier+"-"+architectureIdentifier, symflowerInstallPath); err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot download to %s", symflowerInstallPath)))
	}
	if _, _, err := util.CommandWithResult(logger, &util.Command{
		Command: []string{"chmod", "+x", symflowerInstallPath},
	}); err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, fmt.Sprintf("cannot make %s executable", symflowerInstallPath)))
	}

	return nil
}
