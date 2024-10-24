package cleanup

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanup(t *testing.T) {
	type testCase struct {
		Name string

		Functions []func()

		Validate func(t *testing.T)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			Init()

			for _, f := range tc.Functions {
				Register(f)
			}
			Trigger()

			if tc.Validate != nil {
				tc.Validate(t)
			}
		})
	}

	validate(t, &testCase{
		Name: "None",

		Functions: []func(){},
	})
	{
		value := &atomic.Int32{}
		validate(t, &testCase{
			Name: "Single",

			Functions: []func(){
				func() {
					value.Add(1)
				},
			},

			Validate: func(t *testing.T) {
				assert.EqualValues(t, 1, value.Load())
			},
		})
	}
	{
		value := &atomic.Int32{}
		validate(t, &testCase{
			Name: "Multiple",

			Functions: []func(){
				func() {
					value.Add(1)
				},
				func() {
					value.Add(1)
				},
			},

			Validate: func(t *testing.T) {
				assert.EqualValues(t, 2, value.Load())
			},
		})
	}
}
