package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
)

func TestInstallToolsExecute(t *testing.T) {
	temporaryPath := t.TempDir()

	chmodPath, err := exec.LookPath("chmod")
	require.NoError(t, err)
	t.Setenv("PATH", strings.Join([]string{temporaryPath, filepath.Dir(chmodPath)}, string(os.PathListSeparator)))

	t.Run("Tools are not yet installed", func(t *testing.T) {
		symflowerPath, err := exec.LookPath("symflower")
		require.Error(t, err)
		require.Empty(t, symflowerPath)
	})

	t.Run("Install tools for first time which should install all tools", func(t *testing.T) {
		output, err := osutil.Capture(func() {
			Execute([]string{
				"install-tools",
				"--install-tools-path", temporaryPath,
			})
		})
		require.NoError(t, err)

		require.Contains(t, string(output), `Install "symflower" to`)
		symflowerPath, err := exec.LookPath("symflower")
		require.NoError(t, err)
		require.NotEmpty(t, symflowerPath)
	})

	t.Run("Install tools a second time which should install no new tools", func(t *testing.T) {
		output, err := osutil.Capture(func() {
			Execute([]string{
				"install-tools",
				"--install-tools-path", temporaryPath,
			})
		})
		require.NoError(t, err)

		require.NotContains(t, string(output), `Install "symflower" to`)
		symflowerPath, err := exec.LookPath("symflower")
		require.NoError(t, err)
		require.NotEmpty(t, symflowerPath)
	})
}
