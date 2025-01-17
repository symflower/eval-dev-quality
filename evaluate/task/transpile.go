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
	"golang.org/x/exp/maps"
)

// Transpile holds the transpilation task.
type Transpile struct{}

// ArgumentsTranspile holds extra arguments to be used in a query prompt.
type ArgumentsTranspile struct {
	// OriginLanguage holds the language we are transpiling from.
	OriginLanguage language.Language
	// OriginFilePath holds the path for the file containing the source code we want to transpile.
	OriginFilePath string
}

var _ evaltask.Task = (*Transpile)(nil)

// Identifier returns the transpilation task identifier.
func (t *Transpile) Identifier() evaltask.Identifier {
	return IdentifierTranspile
}

// Run transpiles code between languages and runs predefined tests to check if the transpilation was successful.
func (t *Transpile) Run(ctx evaltask.Context) (repositoryAssessment map[string]map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
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

	repositoryAssessment = map[string]map[evaltask.Identifier]metrics.Assessments{}
	for _, packagePath := range packagePaths {
		originFilePathsWithLanguage, stubFilePath, err := t.unpackTranspilerPackage(ctx, taskLogger.Logger, packagePath)
		if err != nil {
			return nil, nil, err
		}

		modelAssessments := metrics.NewAssessments()
		withSymflowerAssessments := metrics.NewAssessments()
		maximumReachableFiles := uint64(len(language.Languages) - 1) // Transpile repositories contain sub-tasks to transpile from every other supported language minus the one we are transpiling to.
		modelAssessments[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
		withSymflowerAssessments[metrics.AssessmentKeyFilesExecutedMaximumReachable] = maximumReachableFiles
		repositoryAssessment[packagePath] = map[evaltask.Identifier]metrics.Assessments{
			IdentifierTranspile:             modelAssessments,
			IdentifierTranspileSymflowerFix: withSymflowerAssessments,
		}

		for originFilePath, originLanguage := range originFilePathsWithLanguage {
			modelAssessmentsForFile := metrics.NewAssessments()
			withSymflowerAssessmentsForFile := modelAssessmentsForFile // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until we successfully applied "symflower fix".

			if err := ctx.Repository.Reset(ctx.Logger); err != nil {
				ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
			}

			modelContext := model.Context{
				Language: ctx.Language,

				RepositoryPath: filepath.Join(ctx.Repository.DataPath(), packagePath),
				FilePath:       stubFilePath,

				Arguments: &ArgumentsTranspile{
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
			} else {
				testsPassing := testResult.TestsPass
				taskLogger.Info("executed tests", "passing", testsPassing, "symflower-fix", false)
				modelAssessmentsForFile.Award(metrics.AssessmentKeyFilesExecuted)
				modelAssessmentsForFile.AwardMultiple(metrics.AssessmentKeyTestsPassing, uint64(testsPassing))
			}

			if ctx.Language.SupportsFix() {
				withSymflowerFixTestResult, processingTime, ps, err := ExecuteWithSymflowerFix(ctx, taskLogger.Logger, filepath.Join(ctx.Repository.DataPath(), packagePath))
				problems = append(problems, ps...)
				if err != nil {
					problems = append(problems, err)
				} else {
					testsPassing := withSymflowerFixTestResult.TestsPass
					taskLogger.Info("executed tests", "passing", testsPassing, "symflower-fix", true)

					// Symflower was able to fix a failure so now update the assessment with the improved results.
					withSymflowerFixAssessments := metrics.NewAssessments()
					withSymflowerFixAssessments[metrics.AssessmentKeyProcessingTime] = processingTime
					withSymflowerFixAssessments.Award(metrics.AssessmentKeyFilesExecuted)
					withSymflowerFixAssessments.AwardMultiple(metrics.AssessmentKeyTestsPassing, uint64(testsPassing))

					withSymflowerAssessmentsForFile = metrics.CombineWithSymflowerFixAssessments(modelAssessmentsForFile, withSymflowerFixAssessments)
				}
			}

			modelAssessments.Add(modelAssessmentsForFile)
			withSymflowerAssessments.Add(withSymflowerAssessmentsForFile)
		}
	}

	return repositoryAssessment, problems, nil
}

// unpackTranspilerPackage returns a set of source file paths and the corresponding language we want to transpile from and also the path to the file that holds the stub.
func (t *Transpile) unpackTranspilerPackage(ctx evaltask.Context, logger *log.Logger, packagePath string) (originFilePathsWithLanguage map[string]language.Language, stubFilePath string, err error) {
	originFilePathsWithLanguage = map[string]language.Language{}
	packagePathAbsolute := filepath.Join(ctx.Repository.DataPath(), packagePath)

	files, err := os.ReadDir(filepath.Join(packagePathAbsolute, "implementation"))
	if err != nil {
		return nil, "", pkgerrors.WithStack(err)
	}

	for _, file := range files {
		originLanguage, ok := language.LanguageByFileExtension[filepath.Ext(file.Name())]
		if !ok {
			return nil, "", pkgerrors.Errorf("the language extension %q is not supported", filepath.Ext(file.Name()))
		}
		originFilePathsWithLanguage[filepath.Join("implementation", file.Name())] = originLanguage
	}

	stubFilePath, err = packageSourceFile(logger, packagePathAbsolute, ctx.Language)
	if err != nil {
		return nil, "", err
	}

	return originFilePathsWithLanguage, stubFilePath, nil
}

// validateTranspileRepository checks if the repository for the "transpile" task is well-formed.
func validateTranspileRepository(logger *log.Logger, repositoryPath string, destinationLanguage language.Language) (err error) {
	logger.Info("validating repository", "path", repositoryPath)

	packagePaths, err := repositoryOnlyHasPackages(repositoryPath)
	if err != nil {
		return err
	}

	for _, packagePath := range packagePaths {
		implementationDirectoryPath := filepath.Join(packagePath, "implementation")
		files, err := os.ReadDir(implementationDirectoryPath)
		if err != nil {
			return pkgerrors.WithStack(err)
		}

		// Ensure that the implementation directory only contains one source file per language.
		encounteredLanguageAndFile := map[string]string{}
		// Check if the implementation directory is well-formed.
		for _, file := range files {
			if file.IsDir() {
				return pkgerrors.Errorf("the implementation directory %q must contain only source code files to transpile, but found one directory: %q", implementationDirectoryPath, file.Name())
			}

			originLanguage, ok := language.LanguageByFileExtension[filepath.Ext(file.Name())]
			if !ok {
				return pkgerrors.Errorf("the language extension %q is not supported", filepath.Ext(file.Name()))
			} else if encounteredFile := encounteredLanguageAndFile[originLanguage.ID()]; encounteredFile != "" {
				return pkgerrors.Errorf("the implementation directory %q must contain only one source file per language, but found at least two %+v", implementationDirectoryPath, []string{encounteredFile, file.Name()})
			}

			if strings.HasSuffix(file.Name(), originLanguage.DefaultTestFileSuffix()) {
				return pkgerrors.Errorf("the implementation directory %q must contain source files, but found a test file %q", implementationDirectoryPath, files[0])
			}
			encounteredLanguageAndFile[originLanguage.ID()] = file.Name()
		}

		if len(encounteredLanguageAndFile) != len(language.Languages)-1 {
			return pkgerrors.Errorf("the implementation directory %q must contain source files for every language to prevent imbalance, but found only a subset %+v", implementationDirectoryPath, maps.Keys(encounteredLanguageAndFile))
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
