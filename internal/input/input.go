// Package input reads keyboard and gamepad state into world.Controls.
package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"bombjack/internal/world"
)

var (
	leftKeys  = []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyA}
	rightKeys = []ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyD}
	downKeys  = []ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyS}
	jumpKeys  = []ebiten.Key{ebiten.KeyZ, ebiten.KeySpace, ebiten.KeyArrowUp, ebiten.KeyW}
	startKeys = []ebiten.Key{ebiten.KeyEnter}
	fireKeys  = []ebiten.Key{ebiten.KeyX}
	upKeys    = []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyW}
	okKeys    = []ebiten.Key{ebiten.KeyZ}
)

func anyHeld(keys []ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func anyPressed(keys []ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// Read samples the current input. Call exactly once per Update.
func Read() world.Controls {
	c := world.Controls{
		Left:        anyHeld(leftKeys),
		Right:       anyHeld(rightKeys),
		Down:        anyHeld(downKeys),
		Jump:        anyHeld(jumpKeys),
		JumpPressed: anyPressed(jumpKeys),
		Start:       anyPressed(startKeys),
		Fire:        anyPressed(fireKeys),
		Up:          anyHeld(upKeys),
		Confirm:     anyPressed(okKeys),
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		held := func(b ebiten.StandardGamepadButton) bool { return ebiten.IsStandardGamepadButtonPressed(id, b) }
		pressed := func(b ebiten.StandardGamepadButton) bool {
			return inpututil.IsStandardGamepadButtonJustPressed(id, b)
		}
		ax := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		c.Left = c.Left || held(ebiten.StandardGamepadButtonLeftLeft) || ax < -0.4
		c.Right = c.Right || held(ebiten.StandardGamepadButtonLeftRight) || ax > 0.4
		c.Down = c.Down || held(ebiten.StandardGamepadButtonLeftBottom)
		c.Jump = c.Jump || held(ebiten.StandardGamepadButtonRightBottom)
		c.JumpPressed = c.JumpPressed || pressed(ebiten.StandardGamepadButtonRightBottom)
		c.Start = c.Start || pressed(ebiten.StandardGamepadButtonCenterRight)
		c.Fire = c.Fire || pressed(ebiten.StandardGamepadButtonRightLeft)
		c.Up = c.Up || held(ebiten.StandardGamepadButtonLeftTop)
		c.Confirm = c.Confirm || pressed(ebiten.StandardGamepadButtonRightBottom)
	}
	return c
}
