# Bomb Jack (Go + Ebitengine) — Implementation Plan

A Bomb Jack–style arcade platformer with 16-bit-style bitmap graphics, 4 levels,
animated sprites, written in Go with Ebitengine v2. This plan is written so that
several coding agents (e.g. Sonnet 5.5) can work in parallel with minimal
merge conflicts. Each workstream owns specific packages.

---

## 1. Reference: the original Bomb Jack (Tehkan, 1984)

| Mechanic | Original behaviour | Our version |
|---|---|---|
| Goal | Collect all **24 bombs** on a single screen | Same |
| Jack | Very high jump; **press jump again in mid-air to stop rising and float slowly down**; steer left/right in the air | Same. Jump height depends on how long jump is held; down cancels the float |
| Lit bombs | After the first bomb is taken, one other bomb's fuse lights. Taking the lit bomb lights the next one in a fixed sequence | Each level defines a `LitOrder` permutation |
| Round bonus | 20/21/22/23 lit bombs → 10k/20k/30k/50k | Same |
| Bonus meter | Collecting bombs (lit worth more) and coins fills a meter. When full, a bouncing **P** ball appears | Same |
| P | Turns all enemies into **coins** for a few seconds; eating them gives escalating points | 100, 200, 300, 500, 800, 1200, 2000. Duration shrinks per level loop |
| B | Bonus multiplier +1 (max ×5) | Same |
| E | Extra life | Same |
| S | Free credit (rare) | Big bonus of 100k plus an extra life |
| Enemies | **Mummies/birds** spawn at the top and walk/fall. When they reach the floor they **transform** into homing birds, bouncing **saucers** or drifting **orbs** | Same 4 types (mummy, bird, saucer, orb) |
| Screens | Backgrounds of the Sphinx, Acropolis, a castle and a city, each with its own platform layout | Our 4 levels: **Egypt, Greece, Castle, City**. Afterwards they loop with higher difficulty |

---

## 2. Technical specification (shared contract — every agent must follow it)

- **Go** ≥ 1.24, module `bombjack`. **Ebitengine** `github.com/hajimehoshi/ebiten/v2` (latest v2).
- **Logical screen** 256×240. That is a 16 px HUD strip on top plus a **256×224 playfield** (`art.FieldW/FieldH`).
  The window is scaled ×3 (768×720) using nearest-neighbour filtering. `Layout()` returns 256×240.
- **Fixed timestep**: 60 TPS. All movement is in pixels per tick (float64), with no `time.Now()` in game logic.
- **Coordinates**: the origin is the playfield's top-left, and an entity position is the top-left of its 16×16 sprite.
  Each hitbox is an inset `Rect` that is smaller than the sprite: Jack is 10×14, enemies 12×12, bombs 10×10.
- **Platform grid**: 8×8 tiles, so the playfield is 32×28 tiles. Platforms are one tile high, solid from above only
  (like the original, Jack can jump up through them).
- **Graphics come only from `internal/art`**. Do not load external image files at runtime.
  `cmd/spritegen` writes PNGs and `assets/preview/index.html` for review only.
- **Art style**: sprites, tiles, pickups and FX are 16-bit pixel art (hard outlines, a limited palette, ordered dither allowed for shading).
  Static **backdrops use smooth gradients** (`Canvas.VGradientSmooth` / `Lerp`), with **no dithering**, so the sprites stand out.
- **Determinism**: a seeded `math/rand/v2` source lives in the world state, so tests can replay runs.
- **Game logic packages must not import ebiten** (`world`, `player`, `enemy`, `level`, `rules`).
  This keeps them unit-testable headless. Only `gfx`, `game`, `audio`, `input` and `cmd` import ebiten.

### Package layout and ownership

```
cmd/bombjack/        main.go: window setup, ebiten.RunGame              (WS0)
cmd/spritegen/       asset preview generator (exists)                   (WS1)
internal/art/        all pixel art: ASCII sprites, procedural BGs (exists)(WS1)
internal/gfx/        art → *ebiten.Image cache, Animator, text/font      (WS1)
internal/input/      Input interface {Left,Right,Jump,JumpPressed,Down,Start}; keyboard+gamepad impl (WS0)
internal/world/      Rect, Vec, Entity base, tile collision, World struct (WS0 defines, WS2 fills)
internal/player/     Jack state machine and physics                      (WS2)
internal/enemy/      mummy/bird/saucer/orb AI, spawner, transform        (WS3)
internal/level/      4 level definitions: platforms, bombs, LitOrder, spawns (WS4)
internal/rules/      scoring, lit-bomb chain, bonus meter, P/B/E/S, lives (WS4)
internal/audio/      procedural chiptune SFX and music (square/noise synth) (WS5)
internal/game/       scenes: Title, Play, RoundClear, GameOver, HighScore; HUD drawing (WS6)
```

### Core interfaces (WS0 writes these first as compiling stubs)

```go
// internal/world
type Vec struct{ X, Y float64 }
type Rect struct{ X, Y, W, H float64 }
func (r Rect) Overlaps(o Rect) bool

type Platform struct{ TX, TY, Len int }          // tile coords
type World struct {
    Tick      int
    Rng       *rand.Rand
    Platforms []Platform
    Player    *player.Jack   // via interface to avoid cycles if needed
    Enemies   []Enemy
    Bombs     []*Bomb
    Pickups   []*Pickup     // P, B, E, S, coins
    Events    []Event       // queued for rules/audio/fx: BombTaken, JackDied, ...
}
type Enemy interface {
    Update(w *World)
    Hitbox() Rect
    Anim() string          // art animation name, e.g. "bird_fly"
    Pos() Vec
    FacingLeft() bool
    Frightened() bool      // true while P is active: draw as "coin"
}
// Collision helper: resolves landing on platforms + floor, walls at x=0/256.
func (w *World) MoveAndCollide(r Rect, vel Vec, solidFromAbove bool) (Rect, Vec, onGround bool)
```

Events are the decoupling layer. Gameplay code only appends events to the list.
`rules`, `audio` and the particle FX consume them, and the list is cleared each tick.

---

## 3. Workstreams

Model suggestion: **Sonnet 5.5** for WS1–WS5, and a stronger model (Opus) for WS0 and WS6 integration and final review.
Every agent must finish with `gofmt -l . && go vet ./... && go build ./... && go test ./...` all clean,
and must not edit packages it does not own. To ask for a contract change, leave a `// CONTRACT:` note and report it.

### Wave 0 — sequential (1 agent, must finish before Wave 1)

**WS0 Scaffold & contracts**
1. `go get github.com/hajimehoshi/ebiten/v2`. Create `cmd/bombjack/main.go` with window title "BOMB JACK GO", scale ×3, 60 TPS.
2. Create every package in the layout above with the interfaces from §2 as compiling stubs.
3. Write `internal/input` with keyboard mapping (arrows/A-D, Z/Space = jump, Enter = start) plus a gamepad. Also a `FakeInput` for tests.
4. Write `internal/game` with a scene-switch skeleton. It must run, show the Egypt background and quit on Esc.
5. Add `Makefile` targets: `run`, `test`, `sprites` (`go run ./cmd/spritegen -out assets`), `lint`.

*Done when:* `go run ./cmd/bombjack` opens a window showing the level 1 background.

### Wave 1 — parallel (5 agents)

**WS1 Graphics runtime & extra art** (`internal/gfx`, `internal/art`)
- `gfx.Sheet`: build `*ebiten.Image`s once at startup from `art.Animations()` and `art.Backgrounds()`, looked up by name.
- `gfx.Animator{Name; t float64}` with `Update()` and `Draw(dst, x, y, flipH)`, using the FPS and Loop fields of `art.Anim`.
- Platform drawing: left cap, middle and right cap tiles from `platform_<theme>`.
- New art in the same ASCII/procedural style:
  - an 8×8 bitmap **font** (0-9, A-Z and `-!.:×`)
  - score popup digits
  - a **title logo** ("BOMB JACK" in chunky outlined letters)
  - HUD frame and bonus-meter cells
  - Jack **land** squash frame and **turn** frame
  - mummy **transform** "poof" (reuse `explosion` with a tint)
- Background parallax is optional. If added, split the backgrounds into sky and foreground layers.

**WS2 Player** (`internal/player`, collision in `internal/world`)
- States: `Idle, Run, Jump, Float, Fall, Dying, Dead`.
- Physics constants, which should be tunable in one struct:
  - run 1.5 px/tick
  - jump impulse −6.0 while held for up to 14 ticks, with gravity 0.25 applied after release or the cap
  - Float: pressing jump in mid-air sets vy = 0, then fall speed is capped at 0.6
  - normal max fall 3.5
  - holding Down while floating cancels the float
- Air control: full horizontal speed in the air.
- Collisions: platforms are one-way from above, the floor is y = 224−16, and the side walls are solid.
- Death: freeze for 30 ticks, then play `jack_die` and emit `JackDied`.
- Tests: jump apex height, float descent speed, landing on a platform edge, passing up through a platform, not tunnelling at max speed.

**WS3 Enemies** (`internal/enemy`)
- **Mummy**: spawns from a level spawn point and walks the platforms. It turns at platform edges with 50% probability, otherwise it walks off.
  On reaching the floor it plays the transform FX and becomes one of bird/saucer/orb (weights come from the level).
- **Bird**: flies with a homing steer towards Jack (max turn rate 2°/tick, speed 1.0–1.6 depending on level).
- **Saucer**: moves diagonally at constant speed and bounces off walls, floor and ceiling.
- **Orb**: drifts in a lazy sine and occasionally re-targets towards Jack.
- **Spawner**: starts with 2 enemies and adds one every N seconds, up to a per-level cap.
- **Frightened** (P active): enemies freeze and render as `coin`. Touching one emits `CoinEaten` and the enemy respawns later.
- Tests: transform-on-floor, bounce reflection, homing turn limit, and the spawner's cap.

**WS4 Levels & rules** (`internal/level`, `internal/rules`)
- Four `level.Def` structs. Each has a `Theme` index, `Platforms` (tile coords), 24 `Bombs` (pixel positions), a `LitOrder` [24]int,
  `PlayerStart`, `EnemySpawns`, an enemy mix and timings.
  Layouts are inspired by the original, and **City has the fewest platforms** (the original's fifth screen had none).
  Bombs are often placed in rows of 3–4 above platforms and along the walls.
- `rules.State`:
  - score, lives (start 3), multiplier (1–5) and bonus meter (0–100)
  - lit-chain index and lit count, round number, loop
- Scoring:
  - bomb 100×mult, lit bomb 200×mult
  - meter +4 for a bomb and +8 for a lit bomb; when full, spawn P
  - B and E appear at score thresholds, and S at 1/40 probability on P collection
  - round-end lit bonus per the table in §1
- Difficulty per loop: enemy speed +10%, spawn interval −15% and P duration −1 s, with a minimum of 2 s.
- Tests: the full lit chain, the multiplier cap, bonus thresholds and meter→P.

**WS5 Audio** (`internal/audio`)
- A procedural synth at 44.1 kHz with square, pulse, triangle and noise channels. No asset files.
- SFX: jump, float, bomb, lit bomb, P pickup, coin eaten (rising pitch with the combo), death, round clear, extra life.
- A short looping chiptune melody for each level, about 8–16 bars, defined as note data in Go, plus a title jingle.
- Consumes `world.Events`, and has a mute toggle on M.

### Wave 2 — integration (1 agent)

**WS6 Game flow & HUD** (`internal/game`)
- Scenes:
  - **Title**: logo, blinking "PRESS START", high score and attract palette cycling
  - **Play**: an intro "ROUND 1" banner, then play; hit-stop on death
  - **RoundClear**: count the lit bombs, award the bonus, then a wipe transition to the next level
  - **GameOver**: then high-score name entry with 3 letters
- HUD on the 16 px strip: score, high score, lives as Jack icons, the multiplier and the bonus meter.
- FX layer: `sparkle` on bomb pickup, `explosion` on death or transform, and floating score popups.
- High scores are saved as JSON in `os.UserConfigDir()/bombjack/`.
- Pause on P or Esc.

### Wave 3 — QA & polish (1–2 agents)
- Headless soak test that runs `World.Update` for 10 minutes of ticks with random input and checks for no panics or NaNs.
- Balance pass: check that each level can be cleared, and tune enemy speed and spawns.
- Visual pass: check for sprite z-order issues and flicker, and confirm that palette contrast keeps sprites readable on every background.
- Optional: a WASM build (`GOOS=js GOARCH=wasm`) with an HTML page so the game can be played in a browser.

---

## 4. Milestones & acceptance

| # | Milestone | Acceptance |
|---|---|---|
| M0 | Scaffold | The window opens with the background, CI-style checks pass |
| M1 | Jack moves | Jack can run, jump, float and land on level 1 platforms |
| M2 | Core loop | Bombs, lit chain, enemies and death work, and level 1 can be completed |
| M3 | Full game | All 4 levels, P/B/E/S, HUD, title, game over, sound |
| M4 | Polish | Balance, high scores, transitions, soak test green |

## 5. Assets already produced (in this repo)

- `internal/art` contains 22 animations and 4 backgrounds, all drawn in Go code:
  - Jack: idle, run, jump, float and die
  - enemies: mummy, bird, saucer, orb and coin
  - bombs: plain and lit with a sparking fuse
  - P/B/E/S power balls, sparkle and explosion
  - platform tiles for 4 themes
- Regenerate with `go run ./cmd/spritegen -out assets`, then open `assets/preview/index.html`.
- **Adding a sprite**: write a 16×16 ASCII block using the palette runes in `palette.go`, then register it in `library.go`.
  The animation name is the contract used by the game code.

## 6. Progress log

- **2026-10-05 — Waves 0–2 done.** The game is fully playable: title, 4 levels, enemies, lit-bomb chain, P/B/E/S, HUD, FX, synth audio, high score saved.
  - Dev aid: `make shots ROUND=2 TICKS=300,900` runs a scripted autopilot and writes screenshots to `shots/`.
  - **Next is Wave 3 (QA/balance).** The autopilot dies within ~200 ticks, so early-round enemy pressure probably needs softening:
    - a longer spawn-in
    - a slower first mummy
    - spawn points away from Jack
- **2026-10-05 — Wave 3 done. The plan is complete.**
  - **QA:** `internal/sim` adds a headless bot, a 10-minute soak test per level, and a reachability test. The reachability test proves all 24 bombs on every level can be reached.
  - **Balance:** a slower start (first spawn after 200 ticks, 90-tick spawn grace, first mummy at 0.7x speed). Mummies spawn at the point farthest from Jack. Difficulty rises from L1 to L4.
    - Bot clear rate: L1 95%, L2 97%, L3 96%, L4 1%. The low L4 rate is mostly the bot struggling to path to City's high bombs, but L4 needs human playtesting.
  - **Extras:** top-5 high-score table with initials, text panels on title and banners, and a WebAssembly build (`make serve`), verified in a browser.
  - **Open:** human playtest (feel, L4 difficulty), and audio has not been listened to.
- **2026-10-05 — Optional items done.**
  - **Parallax backdrops:** a wider far layer shifts with Jack's position, plus a drifting cloud layer (Egypt, Greece, Castle).
  - **Mummy transform:** a palette-tinted "poof" with a particle shockwave, bandage scraps bouncing on the floor, and rising smoke.
  - **Particle system** (`gfx.FX.Burst`) also used for bombs, pickups, coins, spawns and death.
  - **Title screen:** sparkling stars and star dust on the logo, a logo shine sweep, and colour-cycling text.
  - **Esc** pauses (press again to quit to the title). **I** toggles an invincibility cheat.
- **2026-10-05 — Feel pass.**
  - **Flying:** each mid-air jump tap gives a small lift (`FloatLift`) and a short slow-fall (`FloatHold`). After that, gravity ramps back over `FloatDecay`, so staying up needs repeated tapping.
  - **Bomb pickups:** a cartoon "boom" (32px), trailing flares, bouncing embers, swelling steam puffs, and a screen shake. Lit bombs get a bigger version.
  - **Title screen:** the high-score table fits inside its panel.
- **2026-10-05 — Grouped lit chain.**
  - Levels are defined as bomb groups (rows/columns). LitOrder visits groups in zig-zag order, and bombs within a group in layout order.
  - Taking a bomb lights the *next* bomb after it in LitOrder, so a whole group can be swept in one run or drop.
  - While nothing is lit, the chain-start bomb flashes white after 3 s (`rules.HintBomb`, art `bomb_flash`).
- **2026-10-05 — Homing missiles.**
  - New headless `internal/missile` (a world.System). Each missile follows a cubic Bézier curve whose end tracks the target, advances by arc length while accelerating (`LaunchSpeed` → `MaxSpeed`), and adds a fading sideways wobble. The drawn angle eases toward the motion. It re-targets if its target dies, and salvoes spread across enemies.
  - Rules: 3 missiles at start, max 9, a crate (`PickupM`) every 10 s with one on the field at a time, 500 × multiplier per kill. A crate picked up at 9 missiles gives 1000 pts.
  - FX: an outlined, interpolated smoke trail, and an impact firework (2× boom plus small booms, a rainbow spark ring and spray that crackle, embers, smoke, a white flash and a big shake).
  - Sound: launch whoosh, blast with crackles. X key / gamepad X fires.
- **2026-10-05 — Tweaks.** Losing a life resets the missile stock to 3. In initials entry, X goes back to the previous letter.
- **2026-10-05 — Missile crates give +3** (`MissilesPerBox`), capped at 9. A crate picked up at the cap still pays 1000 pts.
- **2026-10-05 — Initials entry.** Only Z (new `Controls.Confirm`) advances or finishes. Arrows only change the current letter, X goes back, and a 30 s countdown saves the letters on screen when it expires.
- **2026-10-05 — High-score music.** `internal/audio/sid.go` is a 3-voice SID-style tracker running on 50 Hz frames. It uses PWM, 50 Hz arpeggio chords, a resonant filter sweep on the bass, a kick and snare stolen from the bass and chord voices, hard restart, portamento, delayed vibrato and a lead echo. The tune is an original laid-back A-minor funk piece: a 9 s intro, then a 36 s loop.
  - It plays from initials entry through the table, and fades out (1.5 s) into the title music via `audio.FadeTo`.
  - `make music` exports all tracks as WAV files.
- **2026-10-06 — Title pages.** The title panel cycles through four 6 s pages: controls, high scores, "Meet the enemies" (animated, with behaviour hints) and "Bonus items" (a 3×3 grid of animated pickups with their values). Added a `+` glyph to the font.
- **2026-10-06 — Menu and sound test.**
  - Enter/Z on the title opens a menu: Easy / Normal / Hard / Sound & FX. A difficulty is stored in `Game.Difficulty` but not used yet, so every difficulty starts the regular game.
  - The menu returns to the title after 30 s without input, or on Esc.
  - The sound test plays any tune or sound effect (`audio.SoundTest`, `PlayTest`); Esc returns to the title.
  - A nil `*audio.Player` is now valid and silent, for headless tests. Added the `&` and `>` font glyphs.
