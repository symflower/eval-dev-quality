package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/zimmski/osutil/bytesutil"
)

func TestTestDirectiveLinePerFile(t *testing.T) {
	type testCase struct {
		Name string

		Files map[string]string

		Expected map[string]int
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			buffer, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logs:%s", buffer.String())
				}
			}()

			directory := t.TempDir()
			for path, content := range tc.Files {
				require.NoError(t, os.WriteFile(
					filepath.Join(directory, path),
					[]byte(bytesutil.StringTrimIndentations(content)),
					0700,
				))
			}

			actual, err := (&Language{}).testDirectiveLinePerFile(logger, directory)
			require.NoError(t, err)
			assert.Equal(t, tc.Expected, actual)
		})
	}

	validate(t, &testCase{
		Name: "No tests",

		Files: map[string]string{
			"main.rs": `
				fn main() {}
			`,
		},

		Expected: map[string]int{},
	})
	validate(t, &testCase{
		Name: "Single test",

		Files: map[string]string{
			"plain.rs": `
				pub fn plain() {
				    // This does not do anything but it gives us a line to cover.
				}

				#[cfg(test)]
				mod tests {
				    use super::*;

				    #[test]
				    fn test_plain() {
				        // Simply call the function to ensure it runs without panicking
				        plain();
				        // Since plain() doesn't return anything or have observable side effects,
				        // we can only verify that it executes without errors
				    }
				}
			`,
		},

		Expected: map[string]int{
			"plain.rs": 5,
		},
	})
	validate(t, &testCase{
		Name: "Multiple tests",

		Files: map[string]string{
			"plain.rs": `
				pub fn plain() {
				    // This does not do anything but it gives us a line to cover.
				}

				#[cfg(test)]
				mod tests {
				    use super::*;

				    #[test]
				    fn test_plain() {
				        // Simply call the function to ensure it runs without panicking
				        plain();
				        // Since plain() doesn't return anything or have observable side effects,
				        // we can only verify that it executes without errors
				    }
				}
			`,
			"plain_two.rs": `
				pub fn plain() {
				    // This does not do anything but it gives us a line to cover.
				}

				#[cfg(test)]
				mod tests {
				    use super::*;

				    #[test]
				    fn test_plain() {
				        // Simply call the function to ensure it runs without panicking
				        plain();
				        // Since plain() doesn't return anything or have observable side effects,
				        // we can only verify that it executes without errors
				    }
				}
			`,
		},

		Expected: map[string]int{
			"plain.rs":     5,
			"plain_two.rs": 5,
		},
	})
	validate(t, &testCase{
		Name: "Non-rust file",

		Files: map[string]string{
			"README.md": `
				Code example: #[cfg(test)]
			`,
		},

		Expected: map[string]int{},
	})
}

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
