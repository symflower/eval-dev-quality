package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

var claudeEvaluationCSVFileContent = bytesutil.StringTrimIndentations(`
	openrouter/anthropic/claude-2.0,golang,golang/light,write-tests,1,1,1,1,1,1,1,1,1,1
	openrouter/anthropic/claude-2.0,golang,golang/plain,write-tests,2,2,2,2,2,2,2,2,2,2
	openrouter/anthropic/claude-2.0,java,java/light,write-tests,3,3,3,3,3,3,3,3,3,3
	openrouter/anthropic/claude-2.0,java,java/plain,write-tests,4,4,4,4,4,4,4,4,4,4
`)

var gemmaEvaluationCSVFileContent = bytesutil.StringTrimIndentations(`
	openrouter/google/gemma-7b-it,golang,golang/light,write-tests,5,5,5,5,5,5,5,5,5,5
	openrouter/google/gemma-7b-it,golang,golang/plain,write-tests,6,6,6,6,6,6,6,6,6,6
	openrouter/google/gemma-7b-it,java,java/light,write-tests,7,7,7,7,7,7,7,7,7,7
	openrouter/google/gemma-7b-it,java,java/plain,write-tests,8,8,8,8,8,8,8,8,8,8
`)

var gpt4EvaluationCSVFileContent = bytesutil.StringTrimIndentations(`
	openrouter/openai/gpt-4,golang,golang/light,write-tests,9,9,9,9,9,9,9,9,9,9
	openrouter/openai/gpt-4,golang,golang/plain,write-tests,10,10,10,10,10,10,10,10,10,10
	openrouter/openai/gpt-4,java,java/light,write-tests,11,11,11,11,11,11,11,11,11,11
	openrouter/openai/gpt-4,java,java/plain,write-tests,12,12,12,12,12,12,12,12,12,12
`)

func TestReportExecute(t *testing.T) {
	type testCase struct {
		Name string

		Before func(t *testing.T, logger *log.Logger, workingDirectory string)

		Arguments func(workingDirectory string) []string

		ExpectedEvaluationCSVFileContent string
		ExpectedPanicContains            string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			resultPath := filepath.Join(temporaryPath, "result-directory")
			require.NoError(t, osutil.MkdirAll(resultPath))

			logOutput, logger := log.Buffer()
			defer func() {
				log.CloseOpenLogFiles()

				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()

			if tc.Before != nil {
				tc.Before(t, logger, temporaryPath)
			}

			arguments := append([]string{
				"report",
				"--result-path", resultPath,
			}, tc.Arguments(temporaryPath)...)

			if tc.ExpectedPanicContains == "" {
				assert.NotPanics(t, func() {
					Execute(logger, arguments)
				})
			} else {
				didPanic := true
				var recovered any
				func() {
					defer func() {
						recovered = recover()
					}()

					Execute(logger, arguments)

					didPanic = false
				}()
				assert.True(t, didPanic)
				assert.Contains(t, recovered, tc.ExpectedPanicContains)
			}

			file, err := os.ReadFile(filepath.Join(resultPath, "evaluation.csv"))
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedEvaluationCSVFileContent, string(file))
		})
	}

	validate(t, &testCase{
		Name: "Evaluation path does not exist",

		Arguments: func(workingDirectory string) []string {
			return []string{
				"--evaluation-path", filepath.Join("some", "file.csv"),
			}
		},

		ExpectedPanicContains: `the path needs to end with "evaluation.csv"`,
	})
	validate(t, &testCase{
		Name: "Single file",

		Before: func(t *testing.T, logger *log.Logger, workingDirectory string) {
			evaluationFileContent := fmt.Sprintf("%s\n%s", strings.Join(report.EvaluationHeader(), ","), claudeEvaluationCSVFileContent)
			require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, "evaluation.csv"), []byte(evaluationFileContent), 0700))
		},

		Arguments: func(workingDirectory string) []string {
			return []string{
				"--evaluation-path", filepath.Join(workingDirectory, "evaluation.csv"),
			}
		},

		ExpectedEvaluationCSVFileContent: fmt.Sprintf("%s\n%s", strings.Join(report.EvaluationHeader(), ","), claudeEvaluationCSVFileContent),
	})
	validate(t, &testCase{
		Name: "Multiple files",

		Before: func(t *testing.T, logger *log.Logger, workingDirectory string) {
			evaluationFileHeader := strings.Join(report.EvaluationHeader(), ",")
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "claude"), fmt.Sprintf("%s\n%s", evaluationFileHeader, claudeEvaluationCSVFileContent))
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "gemma"), fmt.Sprintf("%s\n%s", evaluationFileHeader, gemmaEvaluationCSVFileContent))
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "openrouter", "gpt4"), fmt.Sprintf("%s\n%s", evaluationFileHeader, gpt4EvaluationCSVFileContent))
		},

		Arguments: func(workingDirectory string) []string {
			return []string{
				"--evaluation-path", filepath.Join(workingDirectory, "docs", "v5", "claude", "evaluation.csv"),
				"--evaluation-path", filepath.Join(workingDirectory, "docs", "v5", "gemma", "evaluation.csv"),
				"--evaluation-path", filepath.Join(workingDirectory, "docs", "v5", "openrouter", "gpt4", "evaluation.csv"),
			}
		},

		ExpectedEvaluationCSVFileContent: fmt.Sprintf("%s\n%s%s%s", strings.Join(report.EvaluationHeader(), ","), claudeEvaluationCSVFileContent, gemmaEvaluationCSVFileContent, gpt4EvaluationCSVFileContent),
	})
	validate(t, &testCase{
		Name: "Multiple files with glob pattern",

		Before: func(t *testing.T, logger *log.Logger, workingDirectory string) {
			evaluationFileHeader := strings.Join(report.EvaluationHeader(), ",")
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "claude"), fmt.Sprintf("%s\n%s", evaluationFileHeader, claudeEvaluationCSVFileContent))
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "gemma"), fmt.Sprintf("%s\n%s", evaluationFileHeader, gemmaEvaluationCSVFileContent))
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "gpt4"), fmt.Sprintf("%s\n%s", evaluationFileHeader, gpt4EvaluationCSVFileContent))
		},

		Arguments: func(workingDirectory string) []string {
			return []string{
				"--evaluation-path", filepath.Join(workingDirectory, "docs", "v5", "*", "evaluation.csv"),
			}
		},

		ExpectedEvaluationCSVFileContent: fmt.Sprintf("%s\n%s%s%s", strings.Join(report.EvaluationHeader(), ","), claudeEvaluationCSVFileContent, gemmaEvaluationCSVFileContent, gpt4EvaluationCSVFileContent),
	})
}

func TestPathsFromGlobPattern(t *testing.T) {
	type testCase struct {
		Name string

		Before func(workingDirectory string)

		EvaluationGlobPattern string

		ExpectedEvaluationFilePaths []string
		ExpectedErrText             string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryDirectory := t.TempDir()

			tc.EvaluationGlobPattern = filepath.Join(temporaryDirectory, tc.EvaluationGlobPattern)

			for i, evaluationFilePath := range tc.ExpectedEvaluationFilePaths {
				tc.ExpectedEvaluationFilePaths[i] = filepath.Join(temporaryDirectory, evaluationFilePath)
			}

			if tc.Before != nil {
				tc.Before(temporaryDirectory)
			}

			actualEvaluationFilePaths, actualErr := pathsFromGlobPattern(tc.EvaluationGlobPattern)
			if len(tc.ExpectedErrText) > 0 {
				assert.ErrorContains(t, actualErr, tc.ExpectedErrText)
			} else {
				require.NoError(t, actualErr)
			}

			assert.ElementsMatch(t, tc.ExpectedEvaluationFilePaths, actualEvaluationFilePaths)
		})
	}

	validate(t, &testCase{
		Name: "File is not an evaluation file",

		Before: func(workingDirectory string) {
			file, err := os.Create(filepath.Join(workingDirectory, "not-an-evaluation.csv"))
			require.NoError(t, err)
			file.Close()
		},

		EvaluationGlobPattern: "not-an-evaluation.csv",

		ExpectedErrText: `the path needs to end with "evaluation.csv"`,
	})
	validate(t, &testCase{
		Name: "Simple",

		Before: func(workingDirectory string) {
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "evaluation.csv"), "")
		},

		EvaluationGlobPattern: "evaluation.csv",

		ExpectedEvaluationFilePaths: []string{
			"evaluation.csv",
		},
	})
	validate(t, &testCase{
		Name: "Glob pattern",

		Before: func(workingDirectory string) {
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "modelA"), "")
			evaluationFileWithContent(t, filepath.Join(workingDirectory, "docs", "v5", "modelB"), "")
		},

		EvaluationGlobPattern: filepath.Join("docs", "v5", "*", "evaluation.csv"),

		ExpectedEvaluationFilePaths: []string{
			filepath.Join("docs", "v5", "modelA", "evaluation.csv"),
			filepath.Join("docs", "v5", "modelB", "evaluation.csv"),
		},
	})
}

func evaluationFileWithContent(t *testing.T, workingDirectory string, content string) {
	require.NoError(t, os.MkdirAll(workingDirectory, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, "evaluation.csv"), []byte(content), 0700))
}