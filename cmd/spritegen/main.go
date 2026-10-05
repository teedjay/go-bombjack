// Command spritegen renders every sprite and background from internal/art to
// PNG files, plus mock level scenes and an animated HTML preview.
//
//	go run ./cmd/spritegen -out assets
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"bombjack/internal/art"
)

type manifestEntry struct {
	Name   string  `json:"name"`
	File   string  `json:"file"`
	W      int     `json:"w"`
	H      int     `json:"h"`
	Frames int     `json:"frames"`
	FPS    float64 `json:"fps"`
	Loop   bool    `json:"loop"`
	PNG    string  `json:"-"`
}

func main() {
	out := flag.String("out", "assets", "output directory")
	flag.Parse()
	must(os.MkdirAll(filepath.Join(*out, "sprites"), 0o755))
	must(os.MkdirAll(filepath.Join(*out, "bg"), 0o755))
	must(os.MkdirAll(filepath.Join(*out, "preview"), 0o755))

	var manifest []manifestEntry
	for _, a := range art.Animations() {
		w, h := a.Size()
		strip := image.NewRGBA(image.Rect(0, 0, w*len(a.Frames), h))
		for i, f := range a.Frames {
			draw.Draw(strip, image.Rect(i*w, 0, (i+1)*w, h), f, image.Point{}, draw.Src)
		}
		file := filepath.Join("sprites", a.Name+".png")
		data := writePNG(filepath.Join(*out, file), strip)
		manifest = append(manifest, manifestEntry{a.Name, file, w, h, len(a.Frames), a.FPS, a.Loop, b64(data)})
	}
	writePNG(filepath.Join(*out, "preview", "contact.png"), contactSheet(art.Animations(), 4))
	j, _ := json.MarshalIndent(manifest, "", "  ")
	must(os.WriteFile(filepath.Join(*out, "sprites.json"), j, 0o644))

	var bgs, scenes []string
	for i, bg := range art.Backgrounds() {
		name := art.Themes[i].Name
		bgs = append(bgs, b64(writePNG(filepath.Join(*out, "bg", name+".png"), bg)))
		scenes = append(scenes, b64(writePNG(filepath.Join(*out, "preview", "scene_"+name+".png"), mockScene(i, bg))))
	}

	logo := b64(writePNG(filepath.Join(*out, "preview", "logo.png"), art.Logo()))
	fontPNG := b64(writePNG(filepath.Join(*out, "preview", "font.png"), fontSheet()))
	var icons []string
	for _, n := range []string{"life", "meter_empty", "meter_full"} {
		icons = append(icons, b64(writePNG(filepath.Join(*out, "sprites", "icon_"+n+".png"), art.Icons()[n])))
	}

	f, err := os.Create(filepath.Join(*out, "preview", "index.html"))
	must(err)
	defer f.Close()
	must(page.Execute(f, map[string]any{"Anims": manifest, "BGs": bgs, "Scenes": scenes, "Themes": art.Themes, "Logo": logo, "Font": fontPNG, "Icons": icons}))
	fmt.Printf("wrote %d animations, %d backgrounds to %s\n", len(manifest), len(bgs), *out)
}

// fontSheet lays out every glyph (with drop shadow) on a dark strip, 2 rows.
func fontSheet() *image.RGBA {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ -!.:x\u00a9"
	rs := []rune(chars)
	per := (len(rs) + 1) / 2
	c := art.NewCanvas(per*8, 16)
	draw.Draw(c, c.Bounds(), &image.Uniform{art.DarkGrey}, image.Point{}, draw.Src)
	for i, r := range rs {
		c.Blit(art.FontShadow()[r], (i%per)*8, (i/per)*8)
	}
	return c.RGBA
}

// mockScene composes a still of what a level will look like in game.
func mockScene(level int, bg *image.RGBA) *image.RGBA {
	c := art.NewCanvas(art.FieldW, art.FieldH)
	c.Blit(bg, 0, 0)
	anims := map[string]art.Anim{}
	for _, a := range art.Animations() {
		anims[a.Name] = a
	}
	tiles := anims["platform_"+art.Themes[level].Name].Frames
	plat := func(tx, ty, n int) {
		for i := 0; i < n; i++ {
			t := tiles[1]
			if i == 0 {
				t = tiles[0]
			} else if i == n-1 {
				t = tiles[2]
			}
			c.Blit(t, (tx+i)*8, ty*8)
		}
	}
	layouts := [][][3]int{
		{{3, 7, 8}, {21, 7, 8}, {11, 13, 10}, {2, 19, 6}, {24, 19, 6}},
		{{12, 6, 8}, {2, 11, 7}, {23, 11, 7}, {12, 17, 8}},
		{{2, 6, 6}, {24, 6, 6}, {8, 12, 16}, {2, 19, 5}, {25, 19, 5}},
		{{5, 8, 6}, {21, 8, 6}, {13, 14, 6}},
	}
	for _, p := range layouts[level] {
		plat(p[0], p[1], p[2])
	}
	bombs := [][2]int{{40, 40}, {60, 40}, {80, 40}, {172, 40}, {192, 40}, {212, 40}, {100, 88}, {120, 88}, {140, 88}, {20, 136}, {230, 136}, {120, 180}}
	for i, b := range bombs {
		name := "bomb"
		if i == 4 {
			name = "bomb_lit"
		}
		c.Blit(anims[name].Frames[0], b[0], b[1])
	}
	c.Blit(anims["jack_float"].Frames[0], 118, 120)
	enemy := []string{"mummy_walk", "bird_fly", "saucer", "orb"}
	c.Blit(anims[enemy[level]].Frames[0], 60, 190)
	c.Blit(anims["bird_fly"].Frames[1], 200, 70)
	c.Blit(anims["orb"].Frames[0], 30, 100)
	c.Blit(anims["power_p"].Frames[0], 180, 150)
	return c.RGBA
}

// contactSheet lays every animation out as a row of frames, scaled up.
func contactSheet(anims []art.Anim, scale int) *image.RGBA {
	const cell = 18
	maxFrames := 0
	for _, a := range anims {
		maxFrames = max(maxFrames, len(a.Frames))
	}
	cols := 2
	rows := (len(anims) + cols - 1) / cols
	w, h := cols*maxFrames*cell+cols*cell, rows*cell
	img := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	draw.Draw(img, img.Bounds(), &image.Uniform{art.DarkGrey}, image.Point{}, draw.Src)
	for i, a := range anims {
		ox := (i % cols) * (maxFrames + 1) * cell
		oy := (i / cols) * cell
		for k, f := range a.Frames {
			b := f.Bounds()
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					p := f.RGBAAt(x, y)
					if p.A == 0 {
						continue
					}
					r := image.Rect((ox+k*cell+1+x)*scale, (oy+1+y)*scale, (ox+k*cell+2+x)*scale, (oy+2+y)*scale)
					draw.Draw(img, r, &image.Uniform{p}, image.Point{}, draw.Src)
				}
			}
		}
	}
	return img
}

func writePNG(path string, img image.Image) []byte {
	var buf bytes.Buffer
	must(png.Encode(&buf, img))
	must(os.WriteFile(path, buf.Bytes(), 0o644))
	return buf.Bytes()
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var page = template.Must(template.New("p").Funcs(template.FuncMap{
	"src": func(s string) template.URL { return template.URL("data:image/png;base64," + s) },
}).Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Bomb Jack Sprites</title>
<style>
:root{--bg:#14101e;--card:#221a33;--ink:#f0e8ff;--muted:#a898c8;--accent:#f8d830}
body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.4 ui-monospace,Menlo,monospace;padding:16px}
h1{color:var(--accent);letter-spacing:2px;font-size:20px}
h2{color:var(--muted);font-size:14px;text-transform:uppercase;letter-spacing:2px;margin-top:32px}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:12px}
.card{background:var(--card);border-radius:8px;padding:10px;text-align:center}
canvas,img{image-rendering:pixelated}
.strip{display:block;margin:6px auto 0;height:24px}
.name{color:var(--accent)}.meta{color:var(--muted);font-size:11px}
.scenes{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:16px}
.scenes img{width:100%;border-radius:6px}
</style></head><body>
<h1>BOMB JACK &mdash; generated sprite set</h1>
<p class="meta">Everything below is drawn from Go code in internal/art (ASCII palette sprites + procedural drawing). Sprites shown at 6&times;.</p>
<h2>Animated sprites</h2>
<div class="grid">
{{range .Anims}}<div class="card">
<canvas width="{{.W}}" height="{{.H}}" style="width:{{.W}}px;height:{{.H}}px;zoom:6" data-src="{{src .PNG}}" data-w="{{.W}}" data-frames="{{.Frames}}" data-fps="{{.FPS}}"></canvas>
<div class="name">{{.Name}}</div><div class="meta">{{.Frames}} frame(s) @ {{.FPS}} fps</div>
<img class="strip" src="{{src .PNG}}">
</div>{{end}}
</div>
<h2>Title logo, font and HUD icons</h2>
<div class="card"><img src="{{src .Logo}}" style="zoom:4"><br><img src="{{src .Font}}" style="zoom:4">
<br>{{range .Icons}}<img src="{{src .}}" style="zoom:6"> {{end}}</div>
<h2>Level mock-ups (2&times;)</h2>
<div class="scenes">{{range $i, $s := .Scenes}}<div><img src="{{src $s}}"><div class="meta">Level {{$i}}</div></div>{{end}}</div>
<h2>Backgrounds (raw 256&times;224)</h2>
<div class="scenes">{{range .BGs}}<img src="{{src .}}">{{end}}</div>
<script>
document.querySelectorAll('canvas[data-src]').forEach(cv=>{
  const ctx=cv.getContext('2d'),img=new Image(),w=+cv.dataset.w,n=+cv.dataset.frames,fps=+cv.dataset.fps||1;
  img.onload=()=>{let f=0;const draw=()=>{ctx.clearRect(0,0,w,cv.height);ctx.drawImage(img,f*w,0,w,cv.height,0,0,w,cv.height);f=(f+1)%n;};draw();if(n>1)setInterval(draw,1000/fps);};
  img.src=cv.dataset.src;
});
</script></body></html>`))
