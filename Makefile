SHELL=/bin/bash
PATH=$(shell go env GOPATH)/bin:$(shell echo $$PATH)

.PHONY: all
all: podman

.PHONY: volume
volume: 
	podman volume inspect dynrouter_go 1>/dev/null 2>&1 || podman volume create dynrouter_go

.PHONY: builder
builder: volume
	podman build -v "$(shell podman volume inspect dynrouter_go  --format "{{.Mountpoint}}"):/go:rw" -t docker.io/pcdummy/go-micro-router-builder:latest -f ./docker/builder/Dockerfile .

.PHONY: run-builder
run-builder: builder
	podman run --rm -v "$(shell echo $$PWD):/code:rw" -v "$(shell podman volume inspect dynrouter_go  --format "{{.Mountpoint}}"):/go:rw" docker.io/pcdummy/go-micro-router-builder:latest $(CMD)

.PHONY: protoc
protoc: builder
	podman run --rm -v "$(shell echo $$PWD):/code:rw" -v "$(shell podman volume inspect dynrouter_go  --format "{{.Mountpoint}}"):/go:rw" docker.io/pcdummy/go-micro-router-builder:latest /bin/sh -c 'cd ./proto/dynrouterpb; protoc --proto_path=/go/bin:. --micro_out=paths=source_relative:. --go_out=paths=source_relative:. dynrouterpb.proto'

.PHONY: podman
podman: protoc
	podman build -v "$(shell echo $$PWD):/code:rw" -v "$(shell podman volume inspect dynrouter_go  --format "{{.Mountpoint}}"):/go:rw" -t docker.io/pcdummy/go-micro-router:latest -f ./docker/go-micro-router/Dockerfile .