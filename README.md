# Bomb Jack Go

A Bomb Jack–style arcade platformer written in Go with [Ebitengine](https://ebitengine.org),
inspired by Tehkan's 1984 arcade classic. Jack leaps and floats around four screens collecting
24 bombs per round, while mummies, birds, saucers and orbs hunt him down.

![In-game screenshot: round 1, Egypt](docs/screenshot.png)

## Features

- **4 levels:** Egypt, Greece, Castle and City. After City they loop with higher difficulty.
- **Classic mechanics:**
  - high jumps, and mid-air **float**
  - the **lit-bomb chain** with a round-end bonus
  - a bonus meter that releases the **P** power ball, which turns enemies into coins
  - **B** (multiplier), **E** (extra life) and **S** (special) power-ups
- **All graphics are generated in Go code** ([internal/art](internal/art)), with no image files:
  - 16-bit-style pixel-art sprites
  - smooth-gradient backdrops
  - a bitmap font
- **Procedural chiptune** sound effects and music ([internal/audio](internal/audio)).
- **Top-5 high-score table** with 3-letter initials, saved locally.
- **Runs in the browser** via a WebAssembly build.

## Controls

| Key | Action |
|---|---|
| ← → (or A / D) | Move |
| Z / Space / ↑ | Jump. Press again in mid-air to **float** |
| ↓ | Cancel float (drop faster) |
| Enter | Start |
| P | Pause |
| M | Mute |
| Esc | Quit |

A standard gamepad also works.

## Build and run

Requirements: **Go 1.27+**.

- **macOS and Windows** need nothing else.
- **Linux** needs the usual Ebitengine dependencies:

```bash
sudo apt install gcc libc6-dev libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config
```

Run directly:

```bash
make run
```

Or build a binary into `bin/`:

```bash
make build
```

Then start it:

```bash
./bin/bombjack
```

Without `make`, use `go run ./cmd/bombjack`.

### Play in the browser (WebAssembly)

```bash
make serve
```

This builds `web/bombjack.wasm` and serves it locally. Then open <http://localhost:8080>.
The `web/` folder is static, so it can be hosted anywhere.

## Development

| Command | Purpose |
|---|---|
| `make test` | Run all unit tests. Game logic is headless and runs without a window |
| `make lint` | gofmt check + `go vet` |
| `make sprites` | Render all sprites/backgrounds to `assets/` plus an animated preview at `assets/preview/index.html` |
| `make shots ROUND=2 TICKS=300,900` | Autopilot run that saves screenshots to `shots/` (`ROUND=-1` = title screen) |
| `make web` / `make serve` | Build or serve the WebAssembly version |

The design, architecture and the multi-agent implementation plan are in [PLAN.md](PLAN.md).
