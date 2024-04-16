package prompt

import (
	"regexp"
	"strings"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/zimmski/osutil/bytesutil"
)

var (
	codeTagRe           = regexp.MustCompile("(^|\n)\\s*```\\w*($|\n)")
	codeTagDuplicatedRe = regexp.MustCompile("```(\\s|\n)*```")
)

// ParseResponse parses code from a model's response.
func ParseResponse(response string) (assessment metrics.Assessments, code string) {
	assessment = metrics.Assessments{}

	// Some models produce duplicated code tags, so unify them if needed.
	response = codeTagDuplicatedRe.ReplaceAllString(response, "```")

	blocks := bytesutil.GuardedBlocks(response, codeTagRe, codeTagRe)

	// When no code blocks are found, assume that just the code is returned.
	if len(blocks) == 0 {
		assessment[metrics.AssessmentKeyNoExcessResponse] = 1

		return assessment, strings.TrimSpace(response)
	}

	// Assume the first code block contains the response code fragment.
	block := blocks[0]

	// Check if the response contained only that single code block.
	responseWithoutBlock := strings.Replace(response, block, "", 1)
	if len(strings.TrimSpace(responseWithoutBlock)) == 0 {
		assessment[metrics.AssessmentKeyNoExcessResponse] = 1
	} else {
		assessment[metrics.AssessmentKeyNoExcessResponse] = 0
	}

	return assessment, strings.TrimSpace(codeTagRe.ReplaceAllString(block, ""))
}
