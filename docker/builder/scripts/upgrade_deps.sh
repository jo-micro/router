#!/bin/bash
set -ex

go mod tidy -go=1.19

for i in $(find . -name 'main.go'); do
    pushd $(dirname $i)
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go get -installsuffix cgo -ldflags="-w -s" -u ./...
    popd
done