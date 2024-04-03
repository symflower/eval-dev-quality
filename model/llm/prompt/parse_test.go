package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"
)

func TestParseResponse(t *testing.T) {
	type testCase struct {
		Name string

		Response string

		ExpectedCode string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualCode := ParseResponse(tc.Response)

			assert.Equal(t, strings.TrimSpace(tc.ExpectedCode), actualCode)
		})
	}

	code := bytesutil.StringTrimIndentations(`
		package main

		import "testing"

		func TestPlain(t *testing.T) {
			plain()
		}
	`)

	validate(t, &testCase{
		Name: "Only Code",

		Response:     code,
		ExpectedCode: code,
	})

	t.Run("Formatted Code", func(t *testing.T) {
		validate(t, &testCase{
			Name: "No Prose",

			Response:     "```\n" + code + "\n```\n",
			ExpectedCode: code,
		})

		validate(t, &testCase{
			Name: "With Prose",

			Response:     "Some text...\n\n```\n" + code + "\n```\n\nSome more text...",
			ExpectedCode: code,
		})
	})

	validate(t, &testCase{
		Name: "Language Specified",

		Response:     "```go\n" + code + "\n```\n",
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Whitespace before Code Block Guards",

		Response:     " ```\n" + code + "\n\t```\n",
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Duplicated Code Block Guards",

		Response:     "```\n```\n" + code + "\n```\n```\n",
		ExpectedCode: code,
	})
}
