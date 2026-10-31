.PHONY: install build run test

install:
	cd web && yarn install

build:
	cd web && yarn build
	go build -o build/repo-server ./cmd/repo

run:
	go run ./cmd/repo serve

test:
	cd web && yarn type-check
	go test ./...
	go vet ./...
