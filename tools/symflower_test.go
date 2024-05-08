package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/zimmski/osutil"
)

func TestSymflowerInstall(t *testing.T) {
	temporaryPath := t.TempDir()

	if osutil.IsWindows() {
		t.Setenv("PATH", temporaryPath)
	} else {
		chmodPath, err := exec.LookPath("chmod")
		require.NoError(t, err)
		t.Setenv("PATH", strings.Join([]string{temporaryPath, filepath.Dir(chmodPath)}, string(os.PathListSeparator)))
	}

	t.Run("Tools are not yet installed", func(t *testing.T) {
		symflowerPath, err := exec.LookPath("symflower" + osutil.BinaryExtension())
		require.Error(t, err)
		require.Empty(t, symflowerPath)
	})

	t.Run("Install tools for first time which should install all tools", func(t *testing.T) {
		logOutput, logger := log.Buffer()
		require.NoError(t, SymflowerInstall(logger, temporaryPath))

		require.Contains(t, logOutput.String(), `Install "symflower" to`)
		symflowerPath, err := exec.LookPath("symflower" + osutil.BinaryExtension())
		require.NoError(t, err)
		require.NotEmpty(t, symflowerPath)
	})

	t.Run("Install tools a second time which should install no new tools", func(t *testing.T) {
		logOutput, logger := log.Buffer()
		require.NoError(t, SymflowerInstall(logger, temporaryPath))

		require.NotContains(t, logOutput.String(), `Install "symflower" to`)
		symflowerPath, err := exec.LookPath("symflower" + osutil.BinaryExtension())
		require.NoError(t, err)
		require.NotEmpty(t, symflowerPath)
	})
}
