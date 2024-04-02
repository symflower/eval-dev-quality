package evaluate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"
)

func TestFormatStringCSV(t *testing.T) {
	type testCase struct {
		Name string

		MetricsPerModel map[string]Metrics

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := FormatStringCSV(tc.MetricsPerModel)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		MetricsPerModel: map[string]Metrics{
			"Model": Metrics{},
		},

		ExpectedString: `
			model,total,executed
			Model,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		MetricsPerModel: map[string]Metrics{
			"ModelA": Metrics{
				Total:    5,
				Executed: 3,
			},
			"ModelB": Metrics{
				Total:    4,
				Executed: 2,
			},
		},

		ExpectedString: `
			model,total,executed
			ModelA,5,3
			ModelB,4,2
		`,
	})
}
