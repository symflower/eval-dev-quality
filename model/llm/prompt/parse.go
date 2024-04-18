package prompt

import (
	"regexp"
	"strings"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/zimmski/osutil/bytesutil"
)

var (
	codeTagMatch           = regexp.MustCompile("(^|\n)\\s*```\\w*($|\n)")
	codeTagDuplicatedMatch = regexp.MustCompile("```(\\s|\n)*```")
)

// ParseResponse parses code from a model's response.
func ParseResponse(response string) (assessment metrics.Assessments, code string) {
	assessment = metrics.Assessments{}

	// Check for empty responses.
	if bytesutil.IsWhitespace(response) {
		return assessment, response
	}
	assessment[metrics.AssessmentKeyResponseNotEmpty]++

	// Some models produce duplicated code tags, so unify them if needed.
	response = codeTagDuplicatedMatch.ReplaceAllString(response, "```")

	blocks := bytesutil.GuardedBlocks(response, codeTagMatch, codeTagMatch)

	// When no code blocks are found, assume that just the code is returned.
	if len(blocks) == 0 {
		// If we cannot distinguish between code and text, we sadly also cannot check if the response contains actual code or if there is any excess response content.

		return assessment, strings.TrimSpace(response)
	}
	assessment[metrics.AssessmentKeyResponseWithCode]++

	// Assume the first code block contains the response code fragment.
	block := blocks[0]

	// Check if the response contained only that single code block.
	responseWithoutBlock := strings.Replace(response, block, "", 1)
	if bytesutil.IsWhitespace(responseWithoutBlock) {
		assessment[metrics.AssessmentKeyResponseNoExcess]++
	}

	return assessment, strings.TrimSpace(codeTagMatch.ReplaceAllString(block, ""))
}
