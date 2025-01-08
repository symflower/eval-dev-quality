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
)

// WriteTests holds the write test task.
type WriteTests struct {
}

var _ evaltask.Task = (*WriteTests)(nil)

// ArgumentsWriteTest holds extra arguments to be used in a query prompt.
type ArgumentsWriteTest struct {
	// Template holds the template data to base the tests onto.
	Template string
	// TestFramework holds the test framework to use.
	TestFramework string
}

// Identifier returns the write test task identifier.
func (t *WriteTests) Identifier() evaltask.Identifier {
	return IdentifierWriteTests
}

// Run generates test files for the given implementation file in a repository.
func (t *WriteTests) Run(ctx evaltask.Context) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	modelCapability, ok := ctx.Model.(model.CapabilityWriteTests)
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

	testFramework := ctx.Language.TestFramework()
	if ctx.Repository.Configuration().Prompt.TestFramework != "" {
		testFramework = ctx.Repository.Configuration().Prompt.TestFramework
	}

	modelAssessment := metrics.NewAssessments()
	withSymflowerFixAssessment := metrics.NewAssessments()
	withSymflowerTemplateAssessment := metrics.NewAssessments()
	withSymflowerTemplateAndFixAssessment := metrics.NewAssessments()

	var maximumReachableFiles uint64
	for _, filePath := range filePaths {
		if ctx.Repository.Configuration().IsFilePathIgnored(filePath) {
			taskLogger.Printf("Ignoring file %q (as configured by the repository)", filePath)

			continue
		}
		maximumReachableFiles++

		// Handle this task case without a template.
		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		arguments := &ArgumentsWriteTest{
			TestFramework: testFramework,
		}
		modelContext := model.Context{
			Language: ctx.Language,

			RepositoryPath: dataPath,
			FilePath:       filePath,

			Logger: taskLogger.Logger,

			Arguments: arguments,
		}
		modelAssessmentFile, withSymflowerFixAssessmentFile, ps, err := runModelAndSymflowerFix(ctx, modelContext, modelCapability.WriteTests)
		problems = append(problems, ps...)
		if err != nil {
			return nil, problems, err
		}
		modelAssessment.Add(modelAssessmentFile)
		withSymflowerFixAssessment.Add(withSymflowerFixAssessmentFile)

		if !ctx.Language.SupportsTemplate() {
			withSymflowerTemplateAssessment.Add(modelAssessmentFile)
			withSymflowerTemplateAndFixAssessment.Add(withSymflowerFixAssessmentFile)

			continue
		}

		// Handle this task case with a template.
		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		_, err = symflowerTemplate(taskLogger.Logger, dataPath, ctx.Language, filePath) // TODO Incorporate template processing time. https://github.com/symflower/eval-dev-quality/issues/350
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, "generating Symflower template"))

			withSymflowerTemplateAssessment.Add(modelAssessmentFile)
			withSymflowerTemplateAndFixAssessment.Add(withSymflowerFixAssessmentFile)

			continue
		}

		testTemplateFilePath := filepath.Join(dataPath, ctx.Language.TestFilePath(dataPath, filePath))
		testTemplate, err := os.ReadFile(testTemplateFilePath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessagef(err, "reading Symflower template from %q", testTemplateFilePath))

			withSymflowerTemplateAssessment.Add(modelAssessmentFile)
			withSymflowerTemplateAndFixAssessment.Add(withSymflowerFixAssessmentFile)

			continue
		}

		arguments.Template = string(testTemplate)
		modelTemplateAssessmentFile, templateWithSymflowerFixAssessmentFile, ps, err := runModelAndSymflowerFix(ctx, modelContext, modelCapability.WriteTests)
		problems = append(problems, ps...)
		if err != nil {
			return nil, problems, err
		}

		withSymflowerTemplateAssessment.Add(modelTemplateAssessmentFile)
		withSymflowerTemplateAndFixAssessment.Add(templateWithSymflowerFixAssessmentFile)
	}

	modelAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerFixAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerTemplateAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerTemplateAndFixAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierWriteTests:                              modelAssessment,
		IdentifierWriteTestsSymflowerFix:                  withSymflowerFixAssessment,
		IdentifierWriteTestsSymflowerTemplate:             withSymflowerTemplateAssessment,
		IdentifierWriteTestsSymflowerTemplateSymflowerFix: withSymflowerTemplateAndFixAssessment,
	}

	return repositoryAssessment, problems, nil
}

// validateWriteTestsRepository checks if the repository for the "write-tests" task is well-formed.
func validateWriteTestsRepository(logger *log.Logger, repositoryPath string, language language.Language) (err error) {
	logger.Printf("validating repository %q", repositoryPath)

	files, err := language.Files(logger, repositoryPath)
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	var sourceFiles []string
	var testFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, language.DefaultTestFileSuffix()) {
			testFiles = append(testFiles, file)
		} else if strings.HasSuffix(file, language.DefaultFileExtension()) {
			sourceFiles = append(sourceFiles, file)
		}
	}

	if len(sourceFiles) == 0 {
		return pkgerrors.Errorf("the repository %q must contain at least one %s source file, but found none", repositoryPath, language.Name())
	} else if len(testFiles) > 0 {
		return pkgerrors.Errorf("the repository %q must contain only %s source files, but found %+v", repositoryPath, language.Name(), testFiles)
	}

	return nil
}
