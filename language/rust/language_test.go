package rust

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil/bytesutil"
)

func TestParseSymflowerTestOutput(t *testing.T) {
	type testCase struct {
		Name string

		Output string

		Total  int
		Passed int
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			total, passed, err := parseSymflowerTestOutput(bytesutil.StringTrimIndentations(tc.Output))
			require.NoError(t, err)
			assert.Equal(t, tc.Total, total)
			assert.Equal(t, tc.Passed, passed)
		})
	}

	validate(t, &testCase{
		Name: "Passing",

		Output: `
			running 1 test
			test plain::tests::test_plain ... ok

			test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
		`,

		Total:  1,
		Passed: 1,
	})
	validate(t, &testCase{
		Name: "Failing",

		Output: `
			running 1 test
			test plain::tests::test_plain ... ok

			test result: FAILED. 0 passed; 1 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
		`,

		Total:  1,
		Passed: 0,
	})
}

func TestMistakes(t *testing.T) {
	type testCase struct {
		Name string

		Files map[string]string

		Expected []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			repository := t.TempDir()
			buffer, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(buffer.String())
				}
			}()

			for path, content := range tc.Files {
				path = filepath.Join(repository, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
				require.NoError(t, os.WriteFile(path, []byte(bytesutil.StringTrimIndentations(content)), 0600))
			}

			actual, err := (&Language{}).Mistakes(logger, repository)
			require.NoError(t, err)
			assert.Equal(t, tc.Expected, actual)
		})
	}

	validate(t, &testCase{
		Name: "No errors",

		Files: map[string]string{
			filepath.Join("src", "lib.rs"): `
				pub fn main(){}
			`,
			"Cargo.toml": `
				[package]
				name = "plain"
				version = "0.1.0"
				edition = "2024"

				[dependencies]
			`,
		},

		Expected: nil,
	})
	validate(t, &testCase{
		Name: "Errors",

		Files: map[string]string{
			filepath.Join("src", "lib.rs"): `
				pub fn main(){
					let x: i32 = "string"; // Compile error: mismatched types
					let y: i32 = 10 / 0; // Compile error: division by zero
				}
			`,
			"Cargo.toml": `
				[package]
				name = "plain"
				version = "0.1.0"
				edition = "2024"

				[dependencies]
			`,
		},

		Expected: []string{
			filepath.Join("src", "lib.rs") + ":2: mismatched types",
		},
	})
}

func TestTestFilePath(t *testing.T) {
	type testCase struct {
		Name string

		SourceFilePath string

		TestFilePath string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t,
				tc.TestFilePath,
				(&Language{}).TestFilePath("", tc.SourceFilePath),
			)
		})
	}

	validate(t, &testCase{
		Name: "Source File",

		SourceFilePath: filepath.Join("src", "foo.rs"),
		TestFilePath:   filepath.Join("tests", "foo_test.rs"),
	})
	validate(t, &testCase{
		Name: "Test File",

		SourceFilePath: filepath.Join("tests", "foo_test.rs"),
		TestFilePath:   filepath.Join("tests", "foo_test.rs"),
	})
}
