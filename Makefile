.PHONY: build
build:
	go build ./cmd/c6

.PHONY: cover
cover:
	go tool cover -html=cover.out

.PHONY: install
install:
	go install ./cmd/c6

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -coverprofile=cover.out -shuffle on ./...

