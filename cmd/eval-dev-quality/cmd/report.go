package cmd

import (
	"os"
	"path/filepath"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	"github.com/symflower/eval-dev-quality/log"
)

// Report holds the "report" command.
type Report struct {
	// EvaluationPaths holds the list of file paths where the results of previous evaluations are stored.
	EvaluationPaths []string `long:"evaluation-path" description:"File path for an evaluation CSV file where the results of a previous evaluation are stored."`
	// ResultPath holds the directory path where the overall results should be written to.
	ResultPath string `long:"result-path" description:"Directory path were the combined results are written to."`

	// logger holds the logger of the command.
	logger *log.Logger
	// timestamp holds the timestamp of the command execution.
	timestamp time.Time
}

var _ SetLogger = (*Evaluate)(nil)

// SetLogger sets the logger of the command.
func (command *Report) SetLogger(logger *log.Logger) {
	command.logger = logger
}

// Execute executes the command.
func (command *Report) Execute(args []string) (err error) {
	// Create the result path directory and check if there is already an evaluation CSV file in there.
	var evaluationCSVFile *os.File
	if err = osutil.MkdirAll(filepath.Dir(command.ResultPath)); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}
	if _, err := os.Stat(filepath.Join(command.ResultPath, "evaluation.csv")); err != nil {
		if os.IsNotExist(err) {
			evaluationCSVFile, err = os.Create(filepath.Join(command.ResultPath, "evaluation.csv"))
			if err != nil {
				command.logger.Panicf("ERROR: %s", err)
			}
			defer evaluationCSVFile.Close()
		} else {
			command.logger.Panicf("ERROR: %s", err)
		}
	} else {
		command.logger.Panicf("ERROR: an evaluation CSV file already exists in %s", command.ResultPath)
	}

	// Collect all evaluation CSV file paths.
	allEvaluationPaths := map[string]bool{}
	for _, evaluationPath := range command.EvaluationPaths {
		evaluationFilePaths, err := pathsFromGlobPattern(evaluationPath)
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		for _, evaluationPath := range evaluationFilePaths {
			allEvaluationPaths[evaluationPath] = true
		}
	}

	// Collect all records from the evaluation CSV files.
	records, err := report.RecordsFromEvaluationCSVFiles(maps.Keys(allEvaluationPaths))
	if err != nil {
		command.logger.Panicf("ERROR: %s", err)
	} else if len(records) == 0 {
		command.logger.Printf("no evaluation records found in %+v", command.EvaluationPaths)

		return nil
	}
	report.SortEvaluationRecords(records)

	// Write all records into a single evaluation CSV file.
	evaluationFile, err := report.NewEvaluationFile(evaluationCSVFile)
	if err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}
	if err := evaluationFile.WriteLines(records); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}

	// Write markdown reports.
	assessmentsPerModel, err := report.RecordsToAssessmentsPerModel(records)
	if err != nil {
		return err
	}
	currentDirectory, err := os.Getwd()
	if err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}
	evaluationLogFiles := collectAllEvaluationLogFiles(append(maps.Keys(allEvaluationPaths), currentDirectory))
	if err := (report.Markdown{
		DateTime: command.timestamp,
		Version:  evaluate.Version,
		Revision: evaluate.Revision,

		LogPaths: evaluationLogFiles,
		CSVPath:  "./evaluation.csv",
		SVGPath:  "./categories.svg",

		AssessmentPerModel: assessmentsPerModel,
	}).WriteToFile(filepath.Join(command.ResultPath, "README.md")); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}

	return nil
}

// collectAllEvaluationLogFiles collects all evaluation log file paths.
func collectAllEvaluationLogFiles(evaluationCSVFilePaths []string) (evaluationLogFilePaths []string) {
	for _, evaluationCSVFilePath := range evaluationCSVFilePaths {
		evaluationDirectory := filepath.Dir(evaluationCSVFilePath)
		_, err := os.Stat(filepath.Join(evaluationDirectory, "evaluation.log"))
		if err != nil {
			continue
		}
		filepath.Base(evaluationDirectory)
		evaluationLogFilePaths = append(evaluationLogFilePaths, filepath.Join(filepath.Base(evaluationDirectory), "evaluation.log"))
	}

	return evaluationLogFilePaths
}

// pathsFromGlobPattern returns all evaluation CSV file paths.
func pathsFromGlobPattern(evaluationGlobPattern string) (evaluationFilePaths []string, err error) {
	if filepath.Base(evaluationGlobPattern) != "evaluation.csv" {
		return nil, pkgerrors.WithStack(pkgerrors.Errorf(`the path needs to end with "evaluation.csv", but found %q`, evaluationGlobPattern))
	}

	evaluationGlobFilePaths, err := filepath.Glob(evaluationGlobPattern)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	} else if len(evaluationGlobFilePaths) == 0 {
		return nil, pkgerrors.Errorf("no files matched the pattern %q", evaluationGlobPattern)
	}

	evaluationCSVFilePaths := map[string]bool{}
	for _, evaluationCSVFilePath := range evaluationGlobFilePaths {
		evaluationCSVFilePaths[evaluationCSVFilePath] = true
	}

	return maps.Keys(evaluationCSVFilePaths), nil
}
