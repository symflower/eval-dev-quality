package main

import (
	"encoding/base64"
	"log"
	"os"
	"path/filepath"

	"github.com/zimmski/osutil"
)

func main() {
	// Cache Symflower's license file if we receive its data and the license file path is not yet set. This is helpful in a container environment where most likely only a environment variable is set.
	licenseData := os.Getenv("SYMFLOWER_INTERNAL_LICENSE_FILE")
	if licenseData != "" {
		homePath, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		licensePath := filepath.Join(homePath, ".symflower-license")
		log.Printf("Write license to %s", licensePath)
		if decoded, err := base64.StdEncoding.DecodeString(licenseData); err == nil {
			licenseData = string(decoded)
		}
		if err := os.WriteFile(licensePath, []byte(licenseData), 0600); err != nil {
			panic(err)
		}

		// Forward the path of the license file for future steps of the job.
		f, err := os.OpenFile(os.Getenv("GITHUB_ENV"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()
		if _, err = f.WriteString("SYMFLOWER_INTERNAL_LICENSE_FILE_PATH" + "=" + licensePath + osutil.LineEnding); err != nil {
			panic(err)
		}
	}
}
