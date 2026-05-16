APP_NAME = resilient-proxy
VERSION = v1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "no-git")
LDFLAGS = -X main.AppName=${APP_NAME} main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) resilient-proxy.go

clean:
	rm -f $(APP_NAME)

.PHONY: build clean