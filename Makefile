BIN       := readme-merge
CMD       := ./cmd/$(BIN)
VERSION   := $(shell cat VERSION)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
GOFLAGS   ?=

.PHONY: build install test test-v lint fmt vet prep check clean version release

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN) $(CMD)

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' $(CMD)

test:
	go test ./... -count=1 -race

test-v:
	go test ./... -v -count=1 -race

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

version:
	@echo $(VERSION)

release:
	@test -z "$$(git status --porcelain)" || (echo "error: working tree is dirty" && exit 1)
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
