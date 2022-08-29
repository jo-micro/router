# STEP 1 build executable binary
FROM docker.io/pcdummy/go-micro-router-builder:latest AS builder

# Create appuser (/etc/passwd entry for the runner container)
RUN useradd appuser

VOLUME [ "/go", "/code" ]

WORKDIR /code

# RUN go mod download
ENV GOPATH="/go"

ARG CACHEBUST=1
ARG VERSION=0.0.1-dev0
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -installsuffix cgo -ldflags='-w -s -X "github.com/go-micro/router/config.Version='$VERSION'"' -o /usr/local/bin/microrouterd github.com/go-micro/router/cmd/microrouterd

# STEP 2 build a small image
# start from busybox
FROM busybox

LABEL maintainer="René Jochum <rene@jochum.dev>"

# Copy certs, passwd and binary from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /usr/local/bin/microrouterd /usr/local/bin/microrouterd
RUN chmod +x /usr/local/bin/microrouterd

# Run as appuser
USER appuser

CMD [ "/usr/local/bin/microrouterd" ]