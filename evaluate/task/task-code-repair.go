package task

import (
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

// TaskCodeRepair holds the code repair task.
type TaskCodeRepair struct {
	// ResultPath holds the directory path where results should be written to.
	ResultPath string

	// Language holds the language for which the task should be evaluated.
	Language language.Language
	// Model holds the model which the task should be evaluated.
	Model model.Model

	// Logger holds the logger for this tasks.
	Logger *log.Logger
}

// TaskArgumentsCodeRepair holds extra arguments to be used in a query prompt.
type TaskArgumentsCodeRepair struct {
	// Mistakes holds the list of compilation errors for a package.
	Mistakes []string
}

var _ evaltask.Task = (*TaskCodeRepair)(nil)

// newCodeRepairTask returns a code repair task.
func newCodeRepairTask(logger *log.Logger, resultPath string, model model.Model, language language.Language) (task evaltask.Task) {
	return &TaskCodeRepair{
		ResultPath: resultPath,
		Language:   language,
		Model:      model,
		Logger:     logger,
	}
}

// Identifier returns the code repair task identifier.
func (t *TaskCodeRepair) Identifier() evaltask.Identifier {
	return IdentifierCodeRepair
}

// Run performs source code repairing in a repository with compilation errors.
// This task requires the repository to consist of multiple packages, with each containing one faulty implementation file and a corresponding test file.
func (t *TaskCodeRepair) Run(repository evaltask.Repository) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	log, logClose, err := log.WithFile(t.Logger, filepath.Join(t.ResultPath, string(t.Identifier()), model.CleanModelNameForFileSystem(t.Model.ID()), t.Language.ID(), repository.Name()+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q on task %q using language %q and repository %q", t.Model.ID(), t.Identifier(), t.Language.ID(), repository.Name())
	defer func() {
		log.Printf("Evaluated model %q on task %q using language %q and repository %q: encountered %d problems: %+v", t.Model.ID(), t.Identifier(), t.Language.ID(), repository.Name(), len(problems), problems)
	}()

	var packagePaths []string
	files, err := os.ReadDir(repository.DataPath())
	if err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}
	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") { // Ignore hidden directories.
			packagePaths = append(packagePaths, filepath.Join(repository.DataPath(), file.Name()))
		}
	}

	modelAssessment := metrics.NewAssessments()
	for _, packagePath := range packagePaths {
		if err := repository.Reset(t.Logger); err != nil {
			t.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		sourceFile, mistakes, err := t.unpackCodeRepairPackage(log, packagePath, repository)
		if err != nil {
			return nil, nil, err
		}

		ctx := evaltask.Context{
			Language: t.Language,

			RepositoryPath: packagePath,
			FilePath:       sourceFile,

			Arguments: &TaskArgumentsCodeRepair{
				Mistakes: mistakes,
			},

			Logger: log,
		}
		assessments, err := t.Model.RunTask(ctx, t.Identifier())
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, sourceFile))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", t.Model.ID(), repository.Name())
		}
		modelAssessment.Add(assessments)
		modelAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := t.Language.Execute(log, packagePath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, sourceFile))

			continue
		}
		log.Printf("Executes tests with %d coverage objects", coverage)
		modelAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		modelAssessment.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
	}

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierCodeRepair: modelAssessment,
	}

	return repositoryAssessment, problems, nil
}

// unpackCodeRepairPackage validates a package under test and returns the source file path and the list of compilation errors found.
func (t *TaskCodeRepair) unpackCodeRepairPackage(fileLogger *log.Logger, packagePath string, repository evaltask.Repository) (sourceFilePath string, mistakes []string, err error) {
	mistakes, err = t.Language.Mistakes(t.Logger, packagePath)
	if err != nil {
		return "", nil, pkgerrors.WithStack(err)
	} else if len(mistakes) == 0 {
		return "", nil, pkgerrors.Errorf("package %q in repository %q must contain source files with compilation errors", packagePath, repository.Name())
	}

	filePaths, err := t.Language.Files(fileLogger, packagePath)
	if err != nil {
		return "", nil, pkgerrors.WithStack(err)
	} else if len(filePaths) != 2 {
		return "", nil, pkgerrors.Errorf("package %q in repository %q must only contain an implementation file and the corresponding test file, but found %#v", packagePath, repository.Name(), filePaths)
	}
	var hasTestFile bool
	for _, file := range filePaths {
		if strings.HasSuffix(file, t.Language.DefaultTestFileSuffix()) {
			hasTestFile = true
		} else if filepath.Ext(file) == t.Language.DefaultFileExtension() {
			sourceFilePath = file
		}
	}
	if sourceFilePath == "" {
		return "", nil, pkgerrors.Errorf("package %q in repository %q does not contain a source file", packagePath, repository.Name())
	} else if !hasTestFile {
		return "", nil, pkgerrors.Errorf("package %q in repository %q does not contain a test file", packagePath, repository.Name())
	}

	return sourceFilePath, mistakes, nil
}
