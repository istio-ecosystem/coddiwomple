all: build

build_date := $(shell date +%s)
build_commit := $(shell git rev-parse --short HEAD)
version ?= 0.2-dev

deps: $(DEP)
	@echo "Fetching dependencies..."
	dep ensure -v

$(DEP):
	@echo "dep not found, installing..."
	@go get github.com/golang/dep/cmd/dep


build: cw
cw:
	go build -v -o cw -ldflags \
	    "-X github.com/istio-ecosystem/coddiwomple/cmd.version=$(version) \
	    -X github.com/istio-ecosystem/coddiwomple/cmd.date=$(build_date) \
	    -X github.com/istio-ecosystem/coddiwomple/cmd.commit=$(build_commit)" \
	    ./cmd/cw
	@chmod +x cw

clean:
	@rm cw || true
