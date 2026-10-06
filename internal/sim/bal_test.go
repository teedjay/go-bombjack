package sim

import (
	"fmt"
	"sort"
	"testing"

	"bombjack/internal/level"
	"bombjack/internal/rules"
)

func median(v []int) int { sort.Ints(v); return v[len(v)/2] }

// TestBalanceReport prints bot metrics per level (run with -v, -run Balance).
func TestBalanceReport(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	for round := 0; round < len(level.Levels); round++ {
		var first, tot, bombs []int
		cl, deaths, ticks := 0, 0, 0
		for s := uint64(1); s <= 120; s++ {
			r := Run(round, s, 3600*3)
			first = append(first, r.FirstDeathTick)
			tot = append(tot, r.TicksAlive)
			deaths += r.Deaths
			ticks += r.TicksAlive
			bombs = append(bombs, r.BombsTaken)
			if r.Cleared {
				cl++
			}
		}
		t.Log(fmt.Sprintf("deaths/1000t=%.2f ", 1000*float64(deaths)/float64(ticks)), fmt.Sprintf("L%d firstDeath p25=%d med=%d total med=%d bombs med=%d cleared=%d/120", round+1, pct(first, 4), median(first), median(tot), median(bombs), cl))
	}
}

func pct(v []int, d int) int { sort.Ints(v); return v[len(v)/d] }

// TestDifficultyReport compares the bot across difficulties (run with -v,
// -run DifficultyReport).
func TestDifficultyReport(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	for _, d := range []rules.Difficulty{rules.Easy, rules.Normal, rules.Hard} {
		for round := 0; round < len(level.Levels); round++ {
			var first []int
			deaths, ticks, cl := 0, 0, 0
			const seeds = 40
			for s := uint64(1); s <= seeds; s++ {
				r := RunDiff(round, s, 3600*3, d)
				first = append(first, r.FirstDeathTick)
				deaths += r.Deaths
				ticks += r.TicksAlive
				if r.Cleared {
					cl++
				}
			}
			t.Logf("%-6v L%d first death med %4d ticks  deaths/1000t %.2f  cleared %2d/%d",
				d, round+1, median(first), 1000*float64(deaths)/float64(ticks), cl, seeds)
		}
	}
}
