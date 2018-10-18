all: build

deps: $(DEP)
	@echo "Fetching dependencies..."
	dep ensure -v

$(DEP):
	@echo "dep not found, installing..."
	@go get github.com/golang/dep/cmd/dep


build: cw
cw:
	go build -o cw ./cmd/cw
	@chmod +x cw

install:
	go install ./cmd/cw

clean:
	@rm cw || true
