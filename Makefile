BIN       := readme-merge
CMD       := ./cmd/$(BIN)
GOFLAGS   ?=

.PHONY: build install test lint fmt vet prep check clean

build:
	go build $(GOFLAGS) -o $(BIN) $(CMD)

install:
	go install $(GOFLAGS) $(CMD)

test:
	go test ./... -count=1

test-v:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

prep: fmt vet test

check: build
	./$(BIN) check

clean:
	rm -f $(BIN)
