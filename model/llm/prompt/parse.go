package prompt

import (
	"regexp"
	"strings"

	"github.com/zimmski/osutil/bytesutil"
)

var (
	codeTagRe           = regexp.MustCompile("(^|\n)\\s*```\\w*($|\n)")
	codeTagDuplicatedRe = regexp.MustCompile("```(\\s|\n)*```")
)

// ParseResponse parses code from a model's response.
func ParseResponse(response string) (code string) {
	// Some models produce duplicated code tags, so unify them if needed.
	response = codeTagDuplicatedRe.ReplaceAllString(response, "```")

	blocks := bytesutil.GuardedBlocks(response, codeTagRe, codeTagRe)
	if len(blocks) == 0 { // When no code blocks are found, assume that just the code is returned.
		return strings.TrimSpace(response)
	}
	// Assume the first code block contains the response code fragment.
	block := blocks[0]

	return strings.TrimSpace(codeTagRe.ReplaceAllString(block, ""))
}
