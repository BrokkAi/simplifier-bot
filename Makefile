.PHONY: build test check licenses
build:
	go build -o bin/bsb ./cmd/bsb
test:
	go test -race ./...
licenses:
	python3 scripts/licenses.py
check: test licenses
	go vet ./...
