.PHONY: install build run

install:
	cd web && yarn install

build:
	cd web && yarn build
	go build -o build/repo ./cmd/repo

run:
	go run ./cmd/repo
