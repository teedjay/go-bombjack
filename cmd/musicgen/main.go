// Command musicgen renders the game's music to WAV files for listening
// outside the game.
//
//	go run ./cmd/musicgen -out music
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bombjack/internal/audio"
)

func main() {
	out := flag.String("out", "music", "output directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	songs := append([]*audio.Song{audio.TitleSong, audio.HiScoreSong, audio.GameOverSong}, audio.LevelSongs[:]...)
	tracks := map[string][]float32{}
	for i, s := range songs {
		intro, loop := audio.RenderSong(s)
		name := s.Name + ".wav"
		if i >= 3 {
			name = fmt.Sprintf("level%d_%s.wav", i-2, s.Name)
		}
		tracks[name] = append(append(append([]float32(nil), intro...), loop...), loop...) // intro + loop twice
	}
	for name, mono := range tracks {
		path := filepath.Join(*out, name)
		if err := writeWAV(path, audio.ToPCM(mono)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s  %.1fs\n", path, float64(len(mono))/audio.SampleRate)
	}
}

// writeWAV writes 16-bit stereo PCM (as produced by audio.ToPCM).
func writeWAV(path string, pcm []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	const ch, bits = 2, 16
	hdr := []any{
		[4]byte{'R', 'I', 'F', 'F'}, uint32(36 + len(pcm)), [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(ch), uint32(audio.SampleRate),
		uint32(audio.SampleRate * ch * bits / 8), uint16(ch * bits / 8), uint16(bits),
		[4]byte{'d', 'a', 't', 'a'}, uint32(len(pcm)),
	}
	for _, v := range hdr {
		if err := binary.Write(f, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	_, err = f.Write(pcm)
	return err
}
