package snow

import (
	"math"
	"testing"
)

func run(s *Sim, seconds float64, f Frame) {
	for i := 0; i < int(seconds*30); i++ {
		s.Update(1.0/30, f)
	}
}

func TestPileGrowsAndIsCapped(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AccumRate = 6
	s := New(1920, 1080, cfg, 1)
	f := Frame{Windows: []Window{{ID: 1, Rect: Rect{400, 300, 1400, 900}}}, Ground: 1040}
	run(s, 1, f)
	var before float32
	for _, h := range s.piles[1].h {
		before += h
	}
	run(s, 240, f)
	var after, peak float32
	for _, h := range s.piles[1].h {
		after += h
		peak = max(peak, h)
	}
	if after <= before*1.5 {
		t.Fatalf("pile did not grow: before=%.1f after=%.1f", before, after)
	}
	if peak > float32(cfg.MaxPile)+0.001 {
		t.Fatalf("pile exceeded cap: %.2f > %.2f", peak, cfg.MaxPile)
	}
}

func TestCoveredWindowGetsNoSnow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AccumRate = 6
	s := New(1920, 1080, cfg, 2)
	// B sits in front of A and fully hides A's top edge.
	f := Frame{Windows: []Window{
		{ID: 2, Rect: Rect{0, 100, 1920, 1000}},  // B (front)
		{ID: 1, Rect: Rect{500, 400, 1000, 800}}, // A (behind)
	}}
	s.Update(1.0/30, f)
	var start float32
	for _, h := range s.piles[1].h {
		start += h
	}
	run(s, 120, f)
	var end float32
	for _, h := range s.piles[1].h {
		end += h
	}
	if math.Abs(float64(end-start)) > 0.5 { // relax() only redistributes, so allow float noise
		t.Fatalf("hidden window accumulated snow: %.2f -> %.2f", start, end)
	}
}

func TestPileFollowsWindowAndIsRemoved(t *testing.T) {
	s := New(1920, 1080, DefaultConfig(), 3)
	s.Update(1.0/30, Frame{Windows: []Window{{ID: 7, Rect: Rect{100, 200, 700, 600}}}})
	if s.piles[7] == nil {
		t.Fatal("no pile created")
	}
	s.Update(1.0/30, Frame{Windows: []Window{{ID: 7, Rect: Rect{900, 300, 1500, 700}}}})
	if s.piles[7] == nil {
		t.Fatal("pile lost when window moved")
	}
	s.Update(1.0/30, Frame{})
	if len(s.piles) != 0 {
		t.Fatal("pile not removed after window closed")
	}
}

func TestFlakesStayFiniteAndOnScreen(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Wind = 200
	s := New(800, 600, cfg, 4)
	run(s, 60, Frame{})
	for _, f := range s.flakes {
		if math.IsNaN(float64(f.x)) || math.IsNaN(float64(f.y)) {
			t.Fatal("NaN flake")
		}
		if f.y > 604 || f.x < -0.2*800-1 || f.x > 1.2*800+1 {
			t.Fatalf("flake escaped: %+v", f)
		}
	}
}

func TestRenderIsValidPremultiplied(t *testing.T) {
	s := New(640, 360, DefaultConfig(), 5)
	f := Frame{Windows: []Window{{ID: 1, Rect: Rect{100, 60, 500, 300}}}, Ground: 330}
	run(s, 30, f)
	c := NewCanvas(640, 360)
	s.Draw(c)
	painted := 0
	for _, p := range c.Pix {
		a, r, g, b := p>>24, p>>16&0xff, p>>8&0xff, p&0xff
		if r > a || g > a || b > a {
			t.Fatalf("pixel %08x is not valid premultiplied alpha", p)
		}
		if a > 0 {
			painted++
		}
	}
	if painted == 0 {
		t.Fatal("nothing was drawn")
	}
}

func TestConfigSanitize(t *testing.T) {
	c := Config{Density: math.NaN(), Wind: 1e9, Snowmen: -5, FPS: 0, MaxPile: 1e6}
	c.Sanitize()
	if c.Density != 0 || c.Wind != 200 || c.Snowmen != 0 || c.FPS != 10 || c.MaxPile != 40 {
		t.Fatalf("bad sanitize: %+v", c)
	}
}

func TestSetConfigLive(t *testing.T) {
	s := New(1000, 1000, DefaultConfig(), 6)
	n := len(s.flakes)
	cfg := DefaultConfig()
	cfg.Density *= 2
	cfg.Snowmen = 0
	s.SetConfig(cfg)
	if len(s.flakes) != 2*n || len(s.men) != 0 {
		t.Fatalf("flakes %d (want %d), men %d", len(s.flakes), 2*n, len(s.men))
	}
}

func TestDraggingShakesSnowOff(t *testing.T) {
	s := New(1920, 1080, DefaultConfig(), 8)
	r := Rect{400, 300, 1000, 700}
	s.Update(1.0/30, Frame{Windows: []Window{{ID: 1, Rect: r}}})
	sum := func() (x float32) {
		for _, h := range s.piles[1].h {
			x += h
		}
		return
	}
	before := sum()
	for i := 1; i <= 5; i++ { // drag the window to the right
		r.L += 40
		r.R += 40
		s.Update(1.0/30, Frame{Windows: []Window{{ID: 1, Rect: r}}})
	}
	if sum() >= before*0.7 {
		t.Fatalf("snow did not fall off: %.1f -> %.1f", before, sum())
	}
	if len(s.puffs) == 0 {
		t.Fatal("no puffs spawned")
	}
	run(s, 3, Frame{Windows: []Window{{ID: 1, Rect: r}}})
	if len(s.puffs) != 0 {
		t.Fatalf("puffs never faded: %d", len(s.puffs))
	}
}
