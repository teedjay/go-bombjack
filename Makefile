.PHONY: run build test lint sprites shots

run:
	go run ./cmd/bombjack

build:
	go build -o bin/bombjack ./cmd/bombjack

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...

sprites:
	go run ./cmd/spritegen -out assets

# Autopilot screenshots: make shots ROUND=0 TICKS=200,600
ROUND ?= 0
TICKS ?= 200,600,1000
shots: build
	BOMBJACK_SHOTS="shots/r$(ROUND):$(ROUND):$(TICKS)" ./bin/bombjack
