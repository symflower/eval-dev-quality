package language

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strconv"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/log"
)

// CoverageBlockUnfolded is an unfolded representation of a coverage data block.
type CoverageBlockUnfolded struct {
	// FileRangeRaw holds the file range string.
	FileRangeRaw string `json:"FileRange"`
	// FilePath holds the file path.
	FilePath string `json:",omitempty"`
	// LineStart holds the start line.
	LineStart int `json:",omitempty"`
	// LineEnd holds the end line.
	LineEnd int `json:",omitempty"`

	// CoverageType holds the covered coverage type.
	CoverageType string
	// Count holds the execution count.
	Count uint
}

// CountUniqueCoverage counts the unique coverage.
func CountUniqueCoverage(data []*CoverageBlockUnfolded) (count int) {
	for _, coverage := range data {
		if coverage.Count > 0 {
			count++
		}
	}

	return count
}

// UniqueCoverageCountFromFile counts the unique coverage listed in the given coverage file.
func UniqueCoverageCountFromFile(logger *log.Logger, coverageFilePath string) (uniqueCoverageCount uint64, err error) {
	coverageData, err := ParseCoverage(logger, coverageFilePath)
	if err != nil {
		return 0, err
	}

	return uint64(CountUniqueCoverage(coverageData)), nil
}

// FileRangeMatch match a textual file range with lines and columns.
var FileRangeMatch = regexp.MustCompile(`^(.+):(\d+):(\d+)-(.+):(\d+):(\d+)$`)

// ParseCoverage parses the given coverage file and returns its coverage.
func ParseCoverage(logger *log.Logger, coverageFilePath string) (coverageData []*CoverageBlockUnfolded, err error) {
	coverageFile, err := os.ReadFile(coverageFilePath)
	if err != nil {
		return nil, pkgerrors.WithMessage(pkgerrors.WithStack(err), coverageFilePath)
	}

	// Log coverage objects.
	logger.Info("coverage objects", "objects", string(coverageFile))

	if err := json.Unmarshal(coverageFile, &coverageData); err != nil {
		return nil, pkgerrors.WithMessage(pkgerrors.WithStack(err), string(coverageFile))
	}
	for _, c := range coverageData {
		fr := FileRangeMatch.FindStringSubmatch(c.FileRangeRaw)
		if fr == nil {
			return nil, pkgerrors.WithMessage(pkgerrors.WithStack(errors.New("could not match file range")), c.FileRangeRaw)
		}
		c.FilePath = fr[1]
		c.LineStart, _ = strconv.Atoi(fr[2]) // The regex guarantees a valid number.
		c.LineEnd, _ = strconv.Atoi(fr[5])   // The regex guarantees a valid number.
	}

	return coverageData, nil
}
