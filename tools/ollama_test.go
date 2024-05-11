package tools

import (
	"testing"

	"github.com/zimmski/osutil"
)

func TestOllamaInstall(t *testing.T) {
	if !osutil.IsLinux() {
		t.Skipf("Installation of Ollama is not supported on this OS")
	}

	ValidateInstallTool(t, NewOllama())
}
