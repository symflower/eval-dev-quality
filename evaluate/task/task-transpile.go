package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

// TaskTranspile holds the transpilation task.
type TaskTranspile struct{}

// TaskArgumentsTranspile holds extra arguments to be used in a query prompt.
type TaskArgumentsTranspile struct {
	// OriginLanguage holds the language we are transpiling from.
	OriginLanguage language.Language
	// OriginFilePath holds the path for the file containing the source code we want to transpile.
	OriginFilePath string
}

var _ evaltask.Task = (*TaskTranspile)(nil)

// Identifier returns the transpilation task identifier.
func (t *TaskTranspile) Identifier() evaltask.Identifier {
	return IdentifierTranspile
}

// Run transpiles code between languages and runs predefined tests to check if the transpilation was successful.
func (t *TaskTranspile) Run(ctx evaltask.Context) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	modelCapability, ok := ctx.Model.(model.CapabilityTranspile)
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

	var packagePaths []string
	files, err := os.ReadDir(ctx.Repository.DataPath())
	if err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}
	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") { // Ignore hidden directories.
			packagePaths = append(packagePaths, file.Name())
		}
	}

	modelAssessments := metrics.NewAssessments()
	withSymflowerAssessments := metrics.NewAssessments()

	maximumReachableFiles := uint64(len(packagePaths))
	modelAssessments[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
	withSymflowerAssessments[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles

	for _, packagePath := range packagePaths {
		modelAssessmentsForFile := metrics.NewAssessments()
		withSymflowerAssessmentsForFile := modelAssessmentsForFile // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until a failure actually happens.

		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		var originLanguage language.Language
		if _, ok := ctx.Language.(*golang.Language); ok {
			originLanguage = &java.Language{}
		} else {
			originLanguage = &golang.Language{}
		}

		originFilePath, stubFilePath, err := t.unpackTranspilerPackage(ctx, taskLogger.Logger, originLanguage, packagePath)
		if err != nil {
			return nil, nil, err
		}

		modelContext := model.Context{
			Language: ctx.Language,

			RepositoryPath: filepath.Join(ctx.Repository.DataPath(), packagePath),
			FilePath:       stubFilePath,

			Arguments: &TaskArgumentsTranspile{
				OriginLanguage: originLanguage,
				OriginFilePath: originFilePath,
			},

			Logger: taskLogger.Logger,
		}
		assessments, err := modelCapability.Transpile(modelContext)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, originFilePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", ctx.Model.ID(), ctx.Repository.Name())
		}
		modelAssessmentsForFile.Add(assessments)
		modelAssessmentsForFile.Award(metrics.AssessmentKeyResponseNoError)

		testResult, ps, err := ctx.Language.ExecuteTests(taskLogger.Logger, filepath.Join(ctx.Repository.DataPath(), packagePath))
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, originFilePath))

			// If there is an execution timeout do not run "symflower fix" because the code itself is correct.
			if errors.Is(err, context.DeadlineExceeded) {
				modelAssessments.Add(modelAssessmentsForFile)
				withSymflowerAssessments.Add(withSymflowerAssessmentsForFile)

				continue
			}

			// Run "symflower fix" if the model response fails to execute.
			if ctx.Language.ID() == "golang" { // Currently we only support Go for "symflower fix".
				withSymflowerFixTestResult, processingTime, ps, err := ExecuteWithSymflowerFix(ctx, taskLogger.Logger, filepath.Join(ctx.Repository.DataPath(), packagePath))
				problems = append(problems, ps...)
				if err != nil {
					problems = append(problems, err)

					modelAssessments.Add(modelAssessmentsForFile)
					withSymflowerAssessments.Add(withSymflowerAssessmentsForFile)

					continue
				} else {
					testsPassing := withSymflowerFixTestResult.TestsPass
					taskLogger.Printf("Executes tests with %d tests passing after \"symflower fix\"", testsPassing)

					// Symflower was able to fix a failure so now update the assessment with the improved results.
					withSymflowerFixAssessments := metrics.NewAssessments()
					withSymflowerFixAssessments[metrics.AssessmentKeyProcessingTime] = processingTime
					withSymflowerFixAssessments.Award(metrics.AssessmentKeyFilesExecuted)
					withSymflowerFixAssessments.AwardPoints(metrics.AssessmentKeyTestsPassing, uint64(testsPassing))

					withSymflowerAssessmentsForFile = metrics.CombineWithSymflowerFixAssessments(modelAssessmentsForFile, withSymflowerFixAssessments)
				}
			}
		} else {
			testsPassing := testResult.TestsPass
			taskLogger.Printf("Executes tests with %d tests passing", testsPassing)
			modelAssessmentsForFile.Award(metrics.AssessmentKeyFilesExecuted)
			modelAssessmentsForFile.AwardPoints(metrics.AssessmentKeyTestsPassing, uint64(testsPassing))
		}

		modelAssessments.Add(modelAssessmentsForFile)
		withSymflowerAssessments.Add(withSymflowerAssessmentsForFile)
	}

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierTranspile:             modelAssessments,
		IdentifierTranspileSymflowerFix: withSymflowerAssessments,
	}

	return repositoryAssessment, problems, nil
}

// unpackTranspilerPackage checks if the testdata repository for the transpilation task is well-formed and returns the path to the implementation file and also the path to the file that holds the stub.
func (t *TaskTranspile) unpackTranspilerPackage(ctx evaltask.Context, fileLogger *log.Logger, originLanguage language.Language, packagePath string) (originFilePath string, stubFilePath string, err error) {
	packagePathAbsolute := filepath.Join(ctx.Repository.DataPath(), packagePath)

	files, err := originLanguage.Files(fileLogger, filepath.Join(packagePathAbsolute, "implementation"))
	if err != nil {
		return "", "", pkgerrors.WithStack(err)
	}
	originFilePath = filepath.Join("implementation", files[0])

	stubFilePath, err = packageSourceFile(fileLogger, packagePathAbsolute, ctx.Language)
	if err != nil {
		return "", "", err
	}

	return originFilePath, stubFilePath, nil
}

// validateTranspileRepository checks if the repository for the "transpile" task is well-formed.
func validateTranspileRepository(logger *log.Logger, repositoryPath string, destinationLanguage language.Language) (err error) {
	logger.Printf("validating repository %q", repositoryPath)

	var originLanguage language.Language
	if _, ok := destinationLanguage.(*golang.Language); ok {
		originLanguage = &java.Language{}
	} else {
		originLanguage = &golang.Language{}
	}

	packagePaths, err := repositoryOnlyHasPackages(repositoryPath)
	if err != nil {
		return err
	}

	for _, packagePath := range packagePaths {
		// Validate the implementation folder.
		files, err := originLanguage.Files(logger, filepath.Join(packagePath, "implementation"))
		if err != nil {
			return pkgerrors.WithStack(err)
		} else if len(files) != 1 {
			return pkgerrors.Errorf("package %q must have an \"implementation\" directory with just one %s source file to transpile", packagePath, originLanguage.Name())
		} else if strings.HasSuffix(files[0], originLanguage.DefaultTestFileSuffix()) {
			return pkgerrors.Errorf("package %q must have an \"implementation\" directory with only a %s source file, but found a test file %q", packagePath, originLanguage.Name(), files[0])
		}

		// Check if the package as one source file and one test file in the language we want to transpile to.
		sourceFiles, testFiles, err := packagesSourceAndTestFiles(logger, packagePath, destinationLanguage)
		if err != nil {
			return err
		} else if len(sourceFiles) != 1 {
			return pkgerrors.Errorf("package %q must contain exactly one %s source file, but found %+v", packagePath, destinationLanguage.Name(), sourceFiles)
		} else if len(testFiles) != 1 {
			return pkgerrors.Errorf("package %q must contain exactly one %s test file, but found %+v", packagePath, destinationLanguage.Name(), testFiles)
		}
	}

	return nil
}
