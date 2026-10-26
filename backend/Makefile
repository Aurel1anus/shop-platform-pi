.PHONY: run test check build import-products

run:
	go run ./cmd/api

test:
	go test ./...

check:
	go vet ./...
	go test ./...

build:
	go build -o bin/api ./cmd/api
	go build -o bin/import-products ./cmd/import-products

import-products:
	go run ./cmd/import-products
