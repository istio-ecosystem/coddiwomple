all: build

deps: $(DEP)
	@echo "Fetching dependencies..."
	dep ensure -v

$(DEP):
	@echo "dep not found, installing..."
	@go get github.com/golang/dep/cmd/dep


build: mcc
mcc:
	go build -o mcc ./cmd/mcc
	@chmod +x mcc

clean:
	@rm mcc
