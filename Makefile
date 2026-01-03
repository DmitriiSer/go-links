BINARY_NAME=go-links
GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)

clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -o $(BINARY_NAME) .

docker:
	docker build -t $(BINARY_NAME) .