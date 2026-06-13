VERSION := $(shell git describe --tags --always --dirty)

.PHONY: all build clean

all: build

build:
	go build -ldflags="-X github.com/alert666/alertmanager-agent/base/conf.agentVersion=$(VERSION)" -o server.exe ./main.go

clean:
	rm -f server.exe
