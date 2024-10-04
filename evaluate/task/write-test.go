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

	modelAssessment := metrics.NewAssessments()
	withSymflowerFixAssessment := metrics.NewAssessments()
	withSymflowerTemplateAssessment := metrics.NewAssessments()
	withSymflowerTemplateAndFixAssessment := metrics.NewAssessments()

	maximumReachableFiles := uint64(len(filePaths))
	modelAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerFixAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerTemplateAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerTemplateAndFixAssessment[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles

	for _, filePath := range filePaths {
		// Handle this task case without a template.
		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		modelAssessmentFile, withSymflowerFixAssessmentFile, ps, err := runModelAndSymflowerFix(ctx, taskLogger, modelCapability, dataPath, filePath, &ArgumentsWriteTest{})
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

			continue
		}

		testTemplateFilePath := filepath.Join(dataPath, ctx.Language.TestFilePath(dataPath, filePath))
		testTemplate, err := os.ReadFile(testTemplateFilePath)
		if err != nil {
			return nil, nil, pkgerrors.WithMessagef(err, "reading Symflower template from %q", testTemplateFilePath)
		}

		modelTemplateAssessmentFile, templateWithSymflowerFixAssessmentFile, ps, err := runModelAndSymflowerFix(ctx, taskLogger, modelCapability, dataPath, filePath, &ArgumentsWriteTest{
			Template: string(testTemplate),
		})
		problems = append(problems, ps...)
		if err != nil {
			return nil, problems, err
		}

		withSymflowerTemplateAssessment.Add(modelTemplateAssessmentFile)
		withSymflowerTemplateAndFixAssessment.Add(templateWithSymflowerFixAssessmentFile)
	}

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierWriteTests:                              modelAssessment,
		IdentifierWriteTestsSymflowerFix:                  withSymflowerFixAssessment,
		IdentifierWriteTestsSymflowerTemplate:             withSymflowerTemplateAssessment,
		IdentifierWriteTestsSymflowerTemplateSymflowerFix: withSymflowerTemplateAndFixAssessment,
	}

	return repositoryAssessment, problems, nil
}

func runModelAndSymflowerFix(ctx evaltask.Context, taskLogger *taskLogger, modelCapability model.CapabilityWriteTests, dataPath string, filePath string, arguments *ArgumentsWriteTest) (modelAssessment metrics.Assessments, withSymflowerFixAssessment metrics.Assessments, problems []error, err error) {
	modelAssessment = metrics.NewAssessments()
	withSymflowerFixAssessment = modelAssessment // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until we successfully applied "symflower fix".
	modelContext := model.Context{
		Language: ctx.Language,

		RepositoryPath: dataPath,
		FilePath:       filePath,

		Logger: taskLogger.Logger,

		Arguments: arguments,
	}
	assessments, err := modelCapability.WriteTests(modelContext)
	if err != nil {
		return nil, nil, append(problems, pkgerrors.WithMessage(err, filePath)), nil
	}
	if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
		return nil, nil, problems, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", ctx.Model.ID(), ctx.Repository.Name())
	}
	modelAssessment.Add(assessments)
	modelAssessment.Award(metrics.AssessmentKeyResponseNoError)

	testResult, ps, err := ctx.Language.ExecuteTests(taskLogger.Logger, dataPath)
	problems = append(problems, ps...)
	if err != nil {
		problems = append(problems, pkgerrors.WithMessage(err, filePath))
	} else {
		taskLogger.Printf("Executes tests with %d coverage objects", testResult.Coverage)
		modelAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		modelAssessment.AwardPoints(metrics.AssessmentKeyCoverage, testResult.Coverage)
	}

	if ctx.Language.SupportsFix() {
		withSymflowerFixTestResult, processingTime, ps, err := ExecuteWithSymflowerFix(ctx, taskLogger.Logger, ctx.Repository.DataPath())
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, err)
		} else {
			ctx.Logger.Printf("with symflower repair: Executes tests with %d coverage objects", withSymflowerFixTestResult.Coverage)

			// Symflower was able to fix a failure so now update the assessment with the improved results.
			withSymflowerFix := metrics.NewAssessments()
			withSymflowerFix[metrics.AssessmentKeyProcessingTime] = processingTime
			withSymflowerFix.Award(metrics.AssessmentKeyFilesExecuted)
			withSymflowerFix.AwardPoints(metrics.AssessmentKeyCoverage, withSymflowerFixTestResult.Coverage)

			withSymflowerFixAssessment = metrics.CombineWithSymflowerFixAssessments(modelAssessment, withSymflowerFix)
		}
	}

	return modelAssessment, withSymflowerFixAssessment, problems, nil
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
