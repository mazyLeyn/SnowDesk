// Command preview renders what SnowDesk looks like to a PNG file, on any OS.
// It is handy for screenshots and for working on the visuals without Windows.
//
//	go run ./cmd/preview -out preview.png
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"snowdesk/snow"
)

func main() {
	out := flag.String("out", "preview.png", "output PNG path")
	secs := flag.Float64("seconds", 150, "simulated seconds (snow piles up over time)")
	icon := flag.String("icon", "", "also write the tray icon (64x64) to this PNG path")
	flag.Parse()

	const W, H = 1280, 720
	cfg := snow.DefaultConfig()
	cfg.AccumRate = 3
	s := snow.New(W, H, cfg, 42)

	// A fake desktop: blue wallpaper, a taskbar and two overlapping windows.
	taskbar := snow.Window{ID: 1, Rect: snow.Rect{L: 0, T: H - 48, R: W, B: H}}
	front := snow.Window{ID: 2, Rect: snow.Rect{L: 330, T: 150, R: 1020, B: 560}}
	back := snow.Window{ID: 3, Rect: snow.Rect{L: 90, T: 250, R: 640, B: 640}}
	frame := snow.Frame{Windows: []snow.Window{front, back, taskbar}, Ground: H - 48}
	for i := 0; i < int(*secs*30); i++ {
		s.Update(1.0/30, frame)
	}

	cv := snow.NewCanvas(W, H)
	s.Draw(cv)

	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			t := float64(y) / H
			img.SetRGBA(x, y, color.RGBA{uint8(20 + 30*t), uint8(50 + 50*t), uint8(150 + 60*t), 255})
		}
	}
	box := func(r snow.Rect, bar, body color.RGBA) {
		for y := r.T; y < r.B; y++ {
			for x := r.L; x < r.R; x++ {
				c := body
				if y < r.T+32 {
					c = bar
				}
				img.SetRGBA(x, y, c)
			}
		}
	}
	box(back.Rect, color.RGBA{44, 44, 52, 255}, color.RGBA{28, 28, 34, 255})
	box(front.Rect, color.RGBA{58, 58, 70, 255}, color.RGBA{32, 32, 40, 255})
	box(taskbar.Rect, color.RGBA{24, 24, 30, 255}, color.RGBA{24, 24, 30, 255})

	for y := 0; y < H; y++ { // composite the premultiplied overlay
		for x := 0; x < W; x++ {
			p := cv.Pix[y*W+x]
			a := p >> 24
			if a == 0 {
				continue
			}
			d := img.RGBAAt(x, y)
			inv := 255 - a
			img.SetRGBA(x, y, color.RGBA{
				uint8(min(255, p>>16&0xff+uint32(d.R)*inv/255)),
				uint8(min(255, p>>8&0xff+uint32(d.G)*inv/255)),
				uint8(min(255, p&0xff+uint32(d.B)*inv/255)),
				255})
		}
	}
	write(*out, img)

	if *icon != "" {
		ic := snow.NewCanvas(64, 64)
		snow.DrawIcon(ic)
		im := image.NewNRGBA(image.Rect(0, 0, 64, 64))
		for i, p := range ic.Pix {
			a := p >> 24
			if a == 0 {
				continue
			}
			un := func(v uint32) uint8 { return uint8(min(255, v*255/a)) }
			im.SetNRGBA(i%64, i/64, color.NRGBA{un(p >> 16 & 0xff), un(p >> 8 & 0xff), un(p & 0xff), uint8(a)})
		}
		write(*icon, im)
	}
}

func write(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
