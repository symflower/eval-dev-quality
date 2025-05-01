package log

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestLogger(t *testing.T) {
	type testCase struct {
		Name string

		Do func(logger *Logger, temporaryPath string)

		ExpectedLogOutputContains []string
		ExpectedFilesContain      map[string][]string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			defer func() {
				CloseOpenLogFiles()
			}()

			logOutput, logger := Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			temporaryPath := t.TempDir()

			tc.Do(logger, temporaryPath)

			actualResultFiles, err := osutil.FilesRecursive(temporaryPath)
			require.NoError(t, err)
			for i, p := range actualResultFiles {
				actualResultFiles[i], err = filepath.Rel(temporaryPath, p)
				require.NoError(t, err)
			}
			sort.Strings(actualResultFiles)
			var expectedResultFiles []string
			for filePath, expectedData := range tc.ExpectedFilesContain {
				expectedResultFiles = append(expectedResultFiles, filePath)

				data, err := os.ReadFile(filepath.Join(temporaryPath, filePath))
				require.NoError(t, err)

				actualData := string(data)
				for _, containsContent := range expectedData {
					assert.Contains(t, actualData, containsContent)
				}
			}
			sort.Strings(expectedResultFiles)
			assert.Equal(t, expectedResultFiles, actualResultFiles)

			for _, expectedLogOutput := range tc.ExpectedLogOutputContains {
				assert.Contains(t, logOutput.String(), expectedLogOutput)
			}
		})
	}

	t.Run("JSON", func(t *testing.T) {
		validate(t, &testCase{
			Name: "New log file for Result path",

			Do: func(logger *Logger, temporaryPath string) {
				logger = logger.With(AttributeKeyResultPath, temporaryPath)

				logger.Info("log-file-content")
			},

			ExpectedLogOutputContains: []string{"log-file-content"},
			ExpectedFilesContain: map[string][]string{
				"evaluation.log": []string{
					"log-file-content",
				},
			},
		})
		validate(t, &testCase{
			Name: "New log file for Language-Model-Repository-Task",

			Do: func(logger *Logger, temporaryPath string) {
				logger = logger.With(AttributeKeyResultPath, temporaryPath)
				logger = logger.With(AttributeKeyLanguage, "languageA")
				logger = logger.With(AttributeKeyModel, "modelA")
				logger = logger.With(AttributeKeyRepository, "repositoryA")
				logger = logger.With(AttributeKeyTask, "taskA")

				logger.Info("log-file-content")
			},

			ExpectedLogOutputContains: []string{"log-file-content"},
			ExpectedFilesContain: map[string][]string{
				"evaluation.log": nil,
				filepath.Join("taskA", "modelA", "languageA", "repositoryA", "evaluation.log"): []string{
					"log-file-content",
				},
			},
		})

		validate(t, &testCase{
			Name: "New log file for two tasks",

			Do: func(logger *Logger, temporaryPath string) {
				logger = logger.With(AttributeKeyResultPath, temporaryPath)
				logger = logger.With(AttributeKeyLanguage, "languageA")
				logger = logger.With(AttributeKeyModel, "modelA")
				logger = logger.With(AttributeKeyRepository, "repositoryA")

				loggerA := logger.With(AttributeKeyTask, "taskA")
				_ = logger.With(AttributeKeyTask, "taskB")

				loggerA.Info("log-file-content-A")
			},

			ExpectedFilesContain: map[string][]string{
				"evaluation.log": nil,
				filepath.Join("taskA", "modelA", "languageA", "repositoryA", "evaluation.log"): []string{
					"log-file-content-A",
				},
				filepath.Join("taskB", "modelA", "languageA", "repositoryA", "evaluation.log"): nil,
			},
		})
		validate(t, &testCase{
			Name: "New log file for two repositories",

			Do: func(logger *Logger, temporaryPath string) {
				logger = logger.With(AttributeKeyResultPath, temporaryPath)
				logger = logger.With(AttributeKeyLanguage, "languageA")
				logger = logger.With(AttributeKeyModel, "modelA")

				loggerA := logger.With(AttributeKeyRepository, "repositoryA")
				_ = loggerA.With(AttributeKeyTask, "taskA")

				loggerB := logger.With(AttributeKeyRepository, "repositoryB")
				_ = loggerB.With(AttributeKeyTask, "taskA")
			},

			ExpectedFilesContain: map[string][]string{
				"evaluation.log": nil,
				filepath.Join("taskA", "modelA", "languageA", "repositoryA", "evaluation.log"): nil,
				filepath.Join("taskA", "modelA", "languageA", "repositoryB", "evaluation.log"): nil,
			},
		})
	})

	t.Run("Text", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Normal log",

			Do: func(logger *Logger, temporaryPath string) {
				logger.Info("log-message")
			},

			ExpectedLogOutputContains: []string{"level=INFO msg=log-message"},
		})
		validate(t, &testCase{
			Name: "Without meta info",

			Do: func(logger *Logger, temporaryPath string) {
				logger.PrintfWithoutMeta("log-message\n")
			},

			ExpectedLogOutputContains: []string{"log-message"},
		})
	})
}

var timeStampRE = regexp.MustCompile(`\d+-\d+-\d+T\d+:\d+:[\d\.]+(Z|[\+\-]\d+:\d+)`)

func TestFileLogger(t *testing.T) {
	type testCase struct {
		Name string

		Log func(t *testing.T, logger *Logger)

		ExpectedContent string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryDirectory := t.TempDir()
			logFilePath := filepath.Join(temporaryDirectory, "log.log")
			logger, loggerClose, err := File(logFilePath)
			require.NoError(t, err)

			tc.Log(t, logger)
			require.NotPanics(t, func() {
				loggerClose()
			})

			logFileContent, err := os.ReadFile(logFilePath)
			require.NoError(t, err)
			logFileContentString := string(logFileContent)

			for _, match := range timeStampRE.FindAllString(logFileContentString, -1) {
				logFileContentString = strings.Replace(logFileContentString, match, "$TIMESTAMP", 1)
			}
			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedContent), logFileContentString)
		})
	}

	validate(t, &testCase{
		Name: "Simple",

		Log: func(t *testing.T, logger *Logger) {
			logger.Info("test")
		},

		ExpectedContent: `
			{"time":"$TIMESTAMP","level":"INFO","msg":"test"}
		`,
	})

	validate(t, &testCase{
		Name: "Error",

		Log: func(t *testing.T, logger *Logger) {
			logger.Info("test", "error", errors.New("error message"))
		},

		ExpectedContent: `
			{"time":"$TIMESTAMP","level":"INFO","msg":"test","error":"error message"}
		`,
	})

	validate(t, &testCase{
		Name: "Error Slice",

		Log: func(t *testing.T, logger *Logger) {
			logger.Info("test", "errors", []error{errors.New("error message"), errors.New("error message")})
		},

		ExpectedContent: `
			{"time":"$TIMESTAMP","level":"INFO","msg":"test","errors":["error message","error message"]}
		`,
	})
}

func TestCleanModelNameForFileSystem(t *testing.T) {
	type testCase struct {
		Name string

		ModelName string

		ExpectedModelNameCleaned string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualModelNameCleaned := CleanModelNameForFileSystem(tc.ModelName)

			assert.Equal(t, tc.ExpectedModelNameCleaned, actualModelNameCleaned)
		})
	}

	validate(t, &testCase{
		Name: "Simple",

		ModelName: "openrouter/anthropic/claude-2.0:beta",

		ExpectedModelNameCleaned: "openrouter_anthropic_claude-2.0_beta",
	})
}
