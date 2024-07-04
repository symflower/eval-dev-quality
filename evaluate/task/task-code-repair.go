package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

// TaskCodeRepair holds the code repair task.
type TaskCodeRepair struct {
}

// TaskArgumentsCodeRepair holds extra arguments to be used in a query prompt.
type TaskArgumentsCodeRepair struct {
	// Mistakes holds the list of compilation errors for a package.
	Mistakes []string
}

var _ evaltask.Task = (*TaskCodeRepair)(nil)

// Identifier returns the code repair task identifier.
func (t *TaskCodeRepair) Identifier() evaltask.Identifier {
	return IdentifierCodeRepair
}

// Run performs source code repairing in a repository with compilation errors.
// This task requires the repository to consist of multiple packages, with each containing one faulty implementation file and a corresponding test file.
func (t *TaskCodeRepair) Run(ctx evaltask.Context) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	modelCapability, ok := ctx.Model.(model.CapabilityRepairCode)
	if !ok {
		pkgerrors.Wrap(evaltask.ErrTaskUnsupportedByModel, fmt.Sprintf("%q does not support %q", ctx.Model.ID(), string(t.Identifier())))
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
			packagePaths = append(packagePaths, filepath.Join(ctx.Repository.DataPath(), file.Name()))
		}
	}

	modelAssessment := metrics.NewAssessments()
	for _, packagePath := range packagePaths {
		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		sourceFile, mistakes, err := t.unpackCodeRepairPackage(ctx, taskLogger.Logger, packagePath)
		if err != nil {
			return nil, nil, err
		}

		modelContext := model.Context{
			Language: ctx.Language,

			RepositoryPath: packagePath,
			FilePath:       sourceFile,

			Arguments: &TaskArgumentsCodeRepair{
				Mistakes: mistakes,
			},

			Logger: taskLogger.Logger,
		}
		assessments, err := modelCapability.RepairCode(modelContext)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, sourceFile))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", ctx.Model.ID(), ctx.Repository.Name())
		}
		modelAssessment.Add(assessments)
		modelAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := ctx.Language.Execute(taskLogger.Logger, packagePath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, sourceFile))

			continue
		}
		taskLogger.Printf("Executes tests with %d coverage objects", coverage)
		modelAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		modelAssessment.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
	}

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierCodeRepair: modelAssessment,
	}

	return repositoryAssessment, problems, nil
}

// unpackCodeRepairPackage validates a package under test and returns the source file path and the list of compilation errors found.
func (t *TaskCodeRepair) unpackCodeRepairPackage(ctx evaltask.Context, fileLogger *log.Logger, packagePath string) (sourceFilePath string, mistakes []string, err error) {
	mistakes, err = ctx.Language.Mistakes(ctx.Logger, packagePath)
	if err != nil {
		return "", nil, pkgerrors.WithStack(err)
	} else if len(mistakes) == 0 {
		return "", nil, pkgerrors.Errorf("package %q in repository %q must contain source files with compilation errors", packagePath, ctx.Repository.Name())
	}

	filePaths, err := ctx.Language.Files(fileLogger, packagePath)
	if err != nil {
		return "", nil, pkgerrors.WithStack(err)
	} else if len(filePaths) != 2 {
		return "", nil, pkgerrors.Errorf("package %q in repository %q must only contain an implementation file and the corresponding test file, but found %#v", packagePath, ctx.Repository.Name(), filePaths)
	}
	var hasTestFile bool
	for _, file := range filePaths {
		if strings.HasSuffix(file, ctx.Language.DefaultTestFileSuffix()) {
			hasTestFile = true
		} else if filepath.Ext(file) == ctx.Language.DefaultFileExtension() {
			sourceFilePath = file
		}
	}
	if sourceFilePath == "" {
		return "", nil, pkgerrors.Errorf("package %q in repository %q does not contain a source file", packagePath, ctx.Repository.Name())
	} else if !hasTestFile {
		return "", nil, pkgerrors.Errorf("package %q in repository %q does not contain a test file", packagePath, ctx.Repository.Name())
	}

	return sourceFilePath, mistakes, nil
}
