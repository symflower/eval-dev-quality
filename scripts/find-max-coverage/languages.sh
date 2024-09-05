#!/bin/bash

echo "java:"
find $1 -path "*/write-tests/*/java/*/*/evaluation.log" | xargs go run scripts/find-max-coverage/main.go
echo "ruby:"
find $1 -path "*/write-tests/*/ruby/*/*/evaluation.log" | xargs go run scripts/find-max-coverage/main.go
echo "golang:"
find $1 -path "*/write-tests/*/golang/*/*/evaluation.log" | xargs go run scripts/find-max-coverage/main.go
