package language

import "regexp"

// CoverageBlockUnfolded is an unfolded representation of a coverage data block.
type CoverageBlockUnfolded struct {
	// FileRange holds the file range.
	FileRange string
	// CoverageType holds the covered coverage type.
	CoverageType string
	// Count holds the execution count.
	Count uint
}

// FileRangeMatch match a textual file range with lines and columns.
var FileRangeMatch = regexp.MustCompile(`^(.+):(\d+):(\d+)-(.+):(\d+):(\d+)$`)
