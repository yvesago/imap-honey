
VERSION=$(shell git describe --abbrev=0 --tags)

BUILD=$(shell git rev-parse --short HEAD)
DATE=$(shell date +%FT%T%z)

# Binaries to be build
PLATFORMS = linux/imaphoney linux/smtphoney
BINS = $(wildcard build/*/*)

# functions
temp = $(subst /, ,$@)
os = $(word 1, $(temp))
target = $(word 2, $(temp))

# Setup the -ldflags option for go building, interpolate the variable values
LDFLAGS=-trimpath -ldflags "-w -s -X 'main.Version=${VERSION}, git: ${BUILD}, build: ${DATE}'"

# Build binaries
#  first build : linux/imaphoney
$(PLATFORMS):
	@mkdir -p build/${os}
	CGO_ENABLED=0 GOARCH=386 GOOS=${os} go build ${LDFLAGS} -o build/$@ ${target}/honey.go
	@echo " => bin builded: build/$@"

build: $(PLATFORMS)

# List binaries
$(BINS):
	@sha256sum $@ 

sha: $(BINS)

# Cleans our project: deletes binaries
clean:
	rm -rf build/
	@echo "Build cleaned"

test:
	go test ./...

all: build

.PHONY: clean build sha $(BINS) $(PLATFORMS)

