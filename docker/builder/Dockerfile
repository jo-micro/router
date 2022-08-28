FROM golang:1.19

# Install packages
RUN apt-get update --allow-releaseinfo-change && \
    apt-get --no-install-recommends --no-install-suggests --yes --quiet install \
        git-core ca-certificates make wget unzip && \
    cd /tmp && wget https://github.com/protocolbuffers/protobuf/releases/download/v21.5/protoc-21.5-linux-x86_64.zip && \
    unzip protoc-21.5-linux-x86_64.zip -d /usr/local/ && chmod +x /usr/local/bin/protoc

# RUN go mod download
ENV GOPATH="/go"

VOLUME [ "/code", "/go" ]

# Install protoc-gen-go + protoc-gen-micro
RUN cd /tmp; go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28; go install github.com/go-micro/generator/cmd/protoc-gen-micro@v1.0.0

WORKDIR /code