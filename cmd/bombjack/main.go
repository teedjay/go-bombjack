package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"bombjack/internal/game"
)

func main() {
	ebiten.SetWindowTitle("BOMB JACK GO")
	ebiten.SetWindowSize(game.ScreenW*3, game.ScreenH*3)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(game.New()); err != nil && !errors.Is(err, game.ErrQuit) {
		log.Fatal(err)
	}
}
