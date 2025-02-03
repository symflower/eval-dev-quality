package cmd

import "os"

func init() {
	// Unset environment variables that are often found as defaults in the terminal configuration.
	if err := os.Unsetenv("PROVIDER_TOKEN"); err != nil {
		panic(err)
	}
	if err := os.Unsetenv("PROVIDER_URL"); err != nil {
		panic(err)
	}
}
