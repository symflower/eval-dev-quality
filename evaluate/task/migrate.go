package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
)

// Migrate holds the migration task.
type Migrate struct{}

var _ evaltask.Task = (*Migrate)(nil)

// ArgumentsMigrate holds extra arguments to be used in a query prompt.
type ArgumentsMigrate struct {
	// TestFramework holds the test framework to use.
	TestFramework string
}

// Identifier returns the migration task identifier.
func (t *Migrate) Identifier() evaltask.Identifier {
	return IdentifierMigrate
}

// Run migrates code and runs the generated tests to check if the migration was successful.
func (t *Migrate) Run(ctx evaltask.Context) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	modelCapability, ok := ctx.Model.(model.CapabilityMigrate)
	if !ok {
		return nil, nil, pkgerrors.Wrap(evaltask.ErrTaskUnsupportedByModel, fmt.Sprintf("%q does not support %q", ctx.Model.ID(), string(t.Identifier())))
	}

	taskLogger, err := newTaskLogger(ctx, t)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		taskLogger.finalize(problems)
	}()

	dataPath := ctx.Repository.DataPath()
	filePaths, err := ctx.Language.Files(taskLogger.Logger, dataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}
	var testFilesPath []string
	for _, filePath := range filePaths {
		if strings.HasSuffix(filePath, ctx.Language.DefaultTestFileSuffix()) {
			testFilesPath = append(testFilesPath, filePath)
		}
	}

	testFramework := ctx.Language.TestFramework()
	if ctx.Repository.Configuration().Prompt.TestFramework != "" {
		testFramework = ctx.Repository.Configuration().Prompt.TestFramework
	}

	modelAssessment := metrics.NewAssessments()
	withSymflowerFixAssessment := metrics.NewAssessments()

	var maximumReachableFiles uint64
	for _, testFilePath := range testFilesPath {
		if ctx.Repository.Configuration().IsFilePathIgnored(testFilePath) {
			taskLogger.Printf("Ignoring file %q (as configured by the repository)", testFilePath)

			continue
		}
		maximumReachableFiles++

		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		// Remove all the other test files so when the tests are executed they don't influence the coverage metrics of the test file under test.
		if err := clearRepositoryForMigration(ctx.Repository.DataPath(), ctx.Language, filePaths, testFilePath); err != nil {
			return nil, nil, err
		}

		modelContext := model.Context{
			Language: ctx.Language,

			RepositoryPath: ctx.Repository.DataPath(),
			FilePath:       testFilePath,

			Arguments: &ArgumentsMigrate{
				TestFramework: testFramework,
			},

			Logger: taskLogger.Logger,
		}
		modelAssessmentFile, withSymflowerFixAssessmentFile, ps, err := runModelAndSymflowerFix(ctx, modelContext, modelCapability.Migrate)
		problems = append(problems, ps...)
		if err != nil {
			return nil, problems, err
		}

		modelAssessment.Add(modelAssessmentFile)
		withSymflowerFixAssessment.Add(withSymflowerFixAssessmentFile)
	}

	modelAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerFixAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierMigrate:             modelAssessment,
		IdentifierMigrateSymflowerFix: withSymflowerFixAssessment,
	}

	return repositoryAssessment, problems, nil
}

// clearRepositoryForMigration removes all files from the repository except the implementation and test file of the given file.
func clearRepositoryForMigration(repositoryPath string, language language.Language, allFilePaths []string, testFilePath string) (err error) {
	testFileName := filepath.Base(testFilePath)
	testFileName = strings.TrimSuffix(testFileName, language.DefaultTestFileSuffix())

	for _, filePath := range allFilePaths {
		if filePath == testFilePath {
			continue
		}

		fileName := filepath.Base(filePath)
		fileName = strings.TrimSuffix(fileName, language.DefaultFileExtension())
		if fileName == testFileName {
			continue
		}

		if err := os.Remove(filepath.Join(repositoryPath, filePath)); err != nil {
			return pkgerrors.WithStack(err)
		}
	}

	return nil
}

// validateTranspileRepository checks if the repository for the "transpile" task is well-formed.
func validateMigrateRepository(logger *log.Logger, repositoryPath string, language language.Language) (err error) {
	logger.Printf("validating repository %q", repositoryPath)

	filePaths, err := osutil.FilesRecursive(repositoryPath)
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	// Keep a mapping between implementation file paths and test file paths.
	implementationFileNames := map[string]bool{}
	testFileNames := map[string]bool{}
	for _, filePath := range filePaths {
		filePathExtension := filepath.Ext(filePath)
		// Ignore build and configuration files.
		if filePathExtension == ".xml" || filePathExtension == ".json" {
			continue
		} else if filePathExtension != language.DefaultFileExtension() {
			return pkgerrors.Errorf("the repository %q must contain only %s files but found %q", repositoryPath, language.Name(), filePath)
		}

		fileName := filepath.Base(filePath)
		if strings.HasSuffix(filePath, language.DefaultTestFileSuffix()) {
			fileName = strings.TrimSuffix(fileName, language.DefaultTestFileSuffix())
			testFileNames[fileName] = true
		} else {
			fileName = strings.TrimSuffix(fileName, language.DefaultFileExtension())
			implementationFileNames[fileName] = true
		}
	}

	if len(implementationFileNames) == 0 {
		return pkgerrors.Errorf("the repository %q must contain implementation files but found none", repositoryPath)
	} else if len(testFileNames) == 0 {
		return pkgerrors.Errorf("the repository %q must contain test files but found none", repositoryPath)
	}

	// Check if for each implementation file a test file exists.
	for implementationFileName := range implementationFileNames {
		if !testFileNames[implementationFileName] {
			return pkgerrors.Errorf("the repository %q must contain a test file for each implementation file but found none for %q", repositoryPath, implementationFileName)
		}
	}

	// Check if for each test file an implementation file exists.
	for testFileName := range testFileNames {
		if !implementationFileNames[testFileName] {
			return pkgerrors.Errorf("the repository %q must contain an implementation file for each test file but found none for %q", repositoryPath, testFileName)
		}
	}

	return nil
}
