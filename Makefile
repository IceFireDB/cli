PROG=bin/cli


SRCS=./cmd

# git commit hash
COMMIT_HASH=$(shell git rev-parse --short HEAD || echo "GitNotFound")

# date
BUILD_DATE=$(shell date '+%Y-%m-%d %H:%M:%S')

# flag
CFLAGS = -ldflags "-s -w -X \"main.BuildVersion=${COMMIT_HASH}\" -X \"main.BuildDate=$(BUILD_DATE)\""

all:
	if [ ! -d "./bin/" ]; then \
	mkdir bin; \
	fi
	go build $(CFLAGS) -o $(PROG) $(SRCS)

# race version
race:
	if [ ! -d "./bin/" ]; then \
    	mkdir bin; \
    	fi
	go build $(CFLAGS) -race -o $(PROG) $(SRCS)

# release version
RELEASE_DATE = $(shell date '+%Y%m%d%H%M%S')
RELEASE_VERSION = $(shell git rev-parse --short HEAD || echo "GitNotFound")
RELEASE_DIR=release_bin
RELEASE_BIN_NAME=IceFireDB
release:
	if [ ! -d "./$(RELEASE_DIR)/$(RELEASE_DATE)_$(RELEASE_VERSION)" ]; then \
	mkdir -p ./$(RELEASE_DIR)/$(RELEASE_DATE)_$(RELEASE_VERSION); \
	fi
	go build  $(CFLAGS) -o $(RELEASE_DIR)/$(RELEASE_DATE)_$(RELEASE_VERSION)/$(RELEASE_BIN_NAME)_linux_amd64 $(SRCS)

install:
	cp $(PROG) $(INSTALL_PREFIX)/bin

	if [ ! -d "${CONF_INSTALL_PREFIX}" ]; then \
	mkdir $(CONF_INSTALL_PREFIX); \
	fi

	cp -R config/* $(CONF_INSTALL_PREFIX)

DLVFLAGS = -ldflags "-X \"main.BuildVersion=${COMMIT_HASH}\" -X \"main.BuildDate=$(BUILD_DATE)\""
DLVGCFLAGS = -gcflags "all=-N -l"
dlv:
	if [ ! -d "./bin/" ]; then \
	mkdir bin; \
	fi
	go build $(DLVFLAGS) $(DLVGCFLAGS) -o $(PROG) $(SRCS)


build: build-proxy build-config all

build-proxy:
	GO111MODULE=on go build -o bin/codis-proxy ./cmd/proxy

build-config:
	GO111MODULE=on go build -o bin/codis-config ./cmd/cconfig

clean:
	rm -rf ./bin

	rm -rf $(INSTALL_PREFIX)/bin

	rm -rf $(CONF_INSTALL_PREFIX)

run:
	go run .

run_dev:
	go run .

test:
	go test -v --v ./...