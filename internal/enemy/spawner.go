package enemy

import "bombjack/internal/world"

const (
	firstMummySlow  = 0.7
	respawnDelay    = 90
	firstSpawnDelay = 200 // quiet time at the start of a round
) // ticks between replacements after an enemy is eaten

// Spawner is a world.System that adds enemies over time. It keeps a target
// population: StartCount at first, +1 every SpawnEvery ticks up to
// MaxEnemies. Eaten enemies are replaced after a short delay.
type Spawner struct {
	Cfg      Config
	started  bool
	target   int
	nextGrow int
	cooldown int
	spawned  int
}

func NewSpawner(cfg Config) *Spawner { return &Spawner{Cfg: cfg} }

func (s *Spawner) spawnOne(w *world.World) {
	pos := world.Vec{X: 120, Y: 0}
	if n := len(s.Cfg.Spawns); n > 0 {
		pos = s.Cfg.Spawns[w.Rng.IntN(n)]
		// prefer the spawn point farthest from Jack
		if w.Player != nil && n > 1 {
			jp := w.Player.Pos()
			best := -1.0
			for _, sp := range s.Cfg.Spawns {
				if d := (sp.X-jp.X)*(sp.X-jp.X) + (sp.Y-jp.Y)*(sp.Y-jp.Y) + w.Rng.Float64(); d > best {
					best, pos = d, sp
				}
			}
		}
	}
	m := NewMummy(w, pos, s.Cfg.Speed, s.Cfg.Mix)
	if s.spawned == 0 {
		m.speed *= firstMummySlow // gentle first enemy of the round
	}
	s.spawned++
	w.Enemies = append(w.Enemies, m)
	w.Emit(world.Event{Kind: world.EvEnemySpawned, Pos: pos})
}

func (s *Spawner) Update(w *world.World) {
	if !s.started {
		s.started = true
		s.target = min(s.Cfg.StartCount, s.Cfg.MaxEnemies)
		s.nextGrow = s.Cfg.SpawnEvery
		s.cooldown = firstSpawnDelay
		if s.Cfg.FirstDelay > 0 {
			s.cooldown = s.Cfg.FirstDelay
		}
	}
	if s.Cfg.SpawnEvery > 0 {
		s.nextGrow--
		if s.nextGrow <= 0 {
			s.nextGrow = s.Cfg.SpawnEvery
			if s.target < s.Cfg.MaxEnemies {
				s.target++
			}
		}
	}
	if s.cooldown > 0 {
		s.cooldown--
	}
	live := 0
	for _, e := range w.Enemies {
		if !e.Removed() {
			live++
		}
	}
	if live < s.target && s.cooldown == 0 {
		s.spawnOne(w)
		s.cooldown = 30
		if live > 0 || w.Tick > 1 {
			s.cooldown = respawnDelay
		}
	}
}
