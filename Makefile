.PHONY: run build test lint sprites shots web serve music

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

# Browser build: make web && make serve, then open http://localhost:8080
web:
	GOOS=js GOARCH=wasm go build -o web/bombjack.wasm ./cmd/bombjack
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

serve: web
	python3 -m http.server 8080 -d web

# Render all music to WAV files in music/ for listening
music:
	go run ./cmd/musicgen -out music
