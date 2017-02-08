#!/bin/bash

set -e

VERSION=0.1
GIT_SHA=$(git rev-parse --short HEAD || echo "GitNotFound")
export CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go build -o bin/pilot ./main.go
