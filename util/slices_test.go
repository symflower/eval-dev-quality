package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertToSortedSlice(t *testing.T) {
	type testCase struct {
		Name string

		Slice []string
		T     string

		ExpectedSlice []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualSlice := InsertToSortedSlice(tc.Slice, tc.T)

			assert.Equal(t, tc.ExpectedSlice, actualSlice)
		})
	}

	validate(t, &testCase{
		Name: "Nil",

		Slice: nil,
		T:     "a",

		ExpectedSlice: []string{"a"},
	})

	validate(t, &testCase{
		Name: "Empty",

		Slice: []string{},
		T:     "a",

		ExpectedSlice: []string{"a"},
	})

	validate(t, &testCase{
		Name: "Insert First",

		Slice: []string{"z"},
		T:     "a",

		ExpectedSlice: []string{"a", "z"},
	})

	validate(t, &testCase{
		Name: "Insert Last",

		Slice: []string{"a"},
		T:     "z",

		ExpectedSlice: []string{"a", "z"},
	})

	validate(t, &testCase{
		Name: "Insert Middle",

		Slice: []string{"a", "c"},
		T:     "b",

		ExpectedSlice: []string{"a", "b", "c"},
	})
}
