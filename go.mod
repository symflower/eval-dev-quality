module github.com/symflower/eval-dev-quality

go 1.21.5

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/jessevdk/go-flags v1.5.1-0.20210607101731-3927b71304df
	github.com/kr/pretty v0.3.1
	github.com/pkg/errors v0.9.1
	github.com/sashabaranov/go-openai v1.20.4
	github.com/stretchr/testify v1.9.0
	github.com/symflower/lockfile v0.0.0-20240419143922-aa3b60940c84
	github.com/zimmski/osutil v1.3.0
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8
	golang.org/x/mod v0.16.0
	gonum.org/v1/gonum v0.15.0 // WORKAROUND v0.15.1 is only supported for Go 1.22+ so explicitly use v0.15.0 to stick with our older Go version.
)

require (
	github.com/Masterminds/semver/v3 v3.3.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/schollz/progressbar/v3 v3.14.2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/yuin/goldmark v1.7.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
