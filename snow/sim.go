// Package snow is the platform-independent heart of SnowDesk: a falling-snow
// simulation, snow piles that build up on window title bars, and a few
// snowmen. It only knows about rectangles and pixels, so it can be tested and
// previewed on any OS; the Windows glue lives in cmd/snowdesk.
package snow

import (
	"math"
	"math/rand"
)

// Rect is a screen rectangle in overlay-local pixels (R and B exclusive).
type Rect struct{ L, T, R, B int }

func (r Rect) W() int { return r.R - r.L }
func (r Rect) H() int { return r.B - r.T }

func (r Rect) contains(x, y float32) bool {
	return x >= float32(r.L) && x < float32(r.R) && y >= float32(r.T) && y < float32(r.B)
}

// Window is a visible top-level window. ID must be stable for the lifetime of
// the window so its snow pile follows it around.
type Window struct {
	ID   uintptr
	Rect Rect
}

// Frame is what the host tells the simulation about the desktop each tick.
// Windows must be ordered from topmost to bottommost.
type Frame struct {
	Windows []Window
	Ground  int // y of the surface snowmen stand on (top of the taskbar)
	StandL  int // x-range snowmen may stand in; both 0 means the whole width
	StandR  int
}

const cell = 2 // pile resolution in px

type flake struct {
	x, y, vy, r, phase, freq, sway, alpha float32
}

type pile struct {
	h    []float32  // accumulated height per cell
	bump []float32  // static smooth noise so the top edge looks lumpy
	ph   [2]float32 // phases of the hill pattern
	last Rect       // where the window was on the previous tick (to detect dragging)
}

// puff is a short-lived clump of snow shaken off a moving window.
type puff struct {
	x, y, vx, vy, r, life, maxLife float32
}

type Sim struct {
	W, H int

	cfg    Config
	rng    *rand.Rand
	t      float64
	wind   float32
	flakes []flake
	puffs  []puff
	piles  map[uintptr]*pile
	men    []snowman
	frame  Frame
}

func New(w, h int, cfg Config, seed int64) *Sim {
	s := &Sim{W: w, H: h, rng: rand.New(rand.NewSource(seed)), piles: map[uintptr]*pile{}}
	s.SetConfig(cfg)
	return s
}

// SetConfig applies new settings live.
func (s *Sim) SetConfig(cfg Config) {
	cfg.Sanitize()
	oldMen := len(s.men)
	s.cfg = cfg
	s.syncFlakeCount()
	if cfg.Snowmen != oldMen {
		s.men = s.newSnowmen(cfg.Snowmen)
	}
	if !cfg.Accumulate {
		s.ClearPiles()
	}
}

// Resize adapts to a new overlay size (display change).
func (s *Sim) Resize(w, h int) {
	s.W, s.H = w, h
	s.syncFlakeCount()
}

// ClearPiles removes all accumulated snow.
func (s *Sim) ClearPiles() { s.piles = map[uintptr]*pile{} }

func (s *Sim) syncFlakeCount() {
	target := int(float64(s.W) * float64(s.H) / 1e6 * s.cfg.Density)
	for len(s.flakes) < target {
		s.flakes = append(s.flakes, s.newFlake(true))
	}
	if len(s.flakes) > target {
		s.flakes = s.flakes[:target]
	}
}

func (s *Sim) newFlake(anywhere bool) flake {
	u := s.rng.Float32()
	r := 0.9 + 2.6*u*u // many small flakes, a few big ones
	f := flake{
		r:     r,
		vy:    28 + r*22 + s.rng.Float32()*10,
		phase: s.rng.Float32() * 2 * math.Pi,
		freq:  0.6 + s.rng.Float32()*1.2,
		sway:  6 + s.rng.Float32()*14,
		alpha: 0.55 + 0.4*(r/3.5),
		x:     (-0.1 + 1.2*s.rng.Float32()) * float32(s.W),
	}
	if anywhere {
		f.y = s.rng.Float32() * float32(s.H)
	} else {
		f.y = -s.rng.Float32() * 12
	}
	return f
}

// eligible reports whether snow can rest on top of w.
func (s *Sim) eligible(w Window) bool {
	return w.Rect.T >= 3 && w.Rect.T < s.H-3 && w.Rect.W() >= 60 && w.Rect.L < s.W && w.Rect.R > 0
}

func (s *Sim) newPile(n int) *pile {
	p := &pile{h: make([]float32, n), bump: make([]float32, n)}
	p.ph = [2]float32{s.rng.Float32() * 6.28, s.rng.Float32() * 6.28}
	s.fillPile(p, 0)
	return p
}

// fillPile initialises cells from index from onward. The surface is a mix of
// broad mounds and smaller lumps, so the snow looks heaped rather than flat.
func (s *Sim) fillPile(p *pile, from int) {
	for i := from; i < len(p.h); i++ {
		p.h[i] = 2 + s.rng.Float32()*1.2
		big := float32(0.5 + 0.5*math.Sin(float64(i)*0.055+float64(p.ph[0])))
		small := float32(0.5 + 0.5*math.Sin(float64(i)*0.16+float64(p.ph[1])))
		p.bump[i] = 11*big*big + 2.4*small + s.rng.Float32()*0.6
	}
}

func (s *Sim) syncPiles(wins []Window) {
	seen := make(map[uintptr]bool, len(wins))
	for _, w := range wins {
		if !s.eligible(w) {
			continue
		}
		seen[w.ID] = true
		n := (w.Rect.W() + cell - 1) / cell
		p := s.piles[w.ID]
		switch {
		case p == nil:
			p = s.newPile(n)
			p.last = w.Rect
			s.piles[w.ID] = p
		case len(p.h) != n: // window was resized
			old := len(p.h)
			p.h = resize(p.h, n)
			p.bump = resize(p.bump, n)
			if n > old {
				s.fillPile(p, old)
			}
		}
		if p.last != w.Rect {
			s.shake(p, w.Rect)
			p.last = w.Rect
		}
	}
	for id := range s.piles {
		if !seen[id] {
			delete(s.piles, id)
		}
	}
}

// shake knocks snow off a window that was just moved or resized: part of the
// pile becomes flying puffs that drift against the direction of motion.
func (s *Sim) shake(p *pile, now Rect) {
	dx, dy := now.L-p.last.L, now.T-p.last.T
	speed := float32(math.Hypot(float64(dx), float64(dy)))
	if speed < 1 {
		speed = 12 // resized: shed a little
	}
	frac := clamp01(speed / 90)
	frac = 0.12 + 0.4*frac
	for i := 0; i < len(p.h); i++ {
		take := (p.h[i] - 0.4) * frac
		if take <= 0 {
			continue
		}
		p.h[i] -= take
		if len(s.puffs) >= 900 {
			continue
		}
		// one puff per ~2 px of removed height keeps the effect dense but cheap
		for n := take * 0.9; n > 0; n-- {
			if n < 1 && s.rng.Float32() > n {
				break
			}
			life := 0.5 + s.rng.Float32()*0.9
			s.puffs = append(s.puffs, puff{
				x:       float32(now.L+i*cell) + s.rng.Float32()*cell,
				y:       float32(now.T) - p.top(i*cell, now.W()) + s.rng.Float32()*2,
				vx:      -float32(dx)*2.2 + (s.rng.Float32()-0.5)*60,
				vy:      -float32(dy)*2.2 - s.rng.Float32()*45,
				r:       0.8 + s.rng.Float32()*1.6,
				life:    life,
				maxLife: life,
			})
		}
	}
}

func resize(a []float32, n int) []float32 {
	if n <= len(a) {
		return a[:n]
	}
	return append(a, make([]float32, n-len(a))...)
}

func (p *pile) cellH(i int) float32 {
	i = max(0, min(i, len(p.h)-1))
	g := p.h[i]
	return g + p.bump[i]*min(1, g/2)
}

// top is the pile height (px above the window's top edge) at relative x.
func (p *pile) top(rel, width int) float32 {
	fx := (float32(rel)+0.5)/cell - 0.5
	i0 := int(math.Floor(float64(fx)))
	t := fx - float32(i0)
	h := p.cellH(i0)*(1-t) + p.cellH(i0+1)*t
	const edge = 8
	d := float32(max(0, min(rel, width-1-rel)))
	if d < edge { // round off the ends of the pile
		h *= float32(math.Sqrt(float64((d + 0.5) / (edge + 0.5))))
	}
	return h
}

// add drops amt of snow near relative x, spread over a few neighbouring cells
// with a triangular kernel so a single flake never makes a spike.
func (p *pile) add(rel int, amt, limit float32) {
	ci := max(0, min(rel/cell, len(p.h)-1))
	for d := -3; d <= 3; d++ {
		if j := ci + d; j >= 0 && j < len(p.h) {
			w := float32(4-max(d, -d)) / 16
			p.h[j] = min(limit, p.h[j]+amt*w)
		}
	}
}

// relax lets the snow settle: each cell drifts toward its neighbours' average.
func (p *pile) relax() {
	var prev float32
	for i := range p.h {
		cur := p.h[i]
		next := cur
		if i+1 < len(p.h) {
			next = p.h[i+1]
		}
		left := prev
		if i == 0 {
			left = cur
		}
		p.h[i] = cur + 0.25*((left+next)/2-cur)
		prev = cur
	}
}

func covered(above []Window, x, y float32) bool {
	for _, w := range above {
		if w.Rect.contains(x, y) {
			return true
		}
	}
	return false
}

// Update advances the simulation by dt seconds.
func (s *Sim) Update(dt float64, f Frame) {
	dt = math.Min(dt, 0.1)
	s.t += dt
	s.frame = f
	s.wind = float32(s.cfg.Wind) * float32(math.Sin(s.t*0.23)+0.6*math.Sin(s.t*0.61+1.3))
	if s.cfg.Accumulate {
		s.syncPiles(f.Windows)
	}
	fdt := float32(dt)
	W, H := float32(s.W), float32(s.H)

	for i := range s.flakes {
		fl := &s.flakes[i]
		oy := fl.y
		fl.x += (s.wind*(0.4+fl.r*0.25) + fl.sway*float32(math.Sin(float64(fl.phase)+s.t*float64(fl.freq)*2))) * fdt
		fl.y += fl.vy * fdt

		landed := false
		if s.cfg.Accumulate {
			for wi, w := range f.Windows {
				if !s.eligible(w) || fl.x < float32(w.Rect.L) || fl.x >= float32(w.Rect.R) {
					continue
				}
				p := s.piles[w.ID]
				if p == nil {
					continue
				}
				rel := int(fl.x) - w.Rect.L
				surf := float32(w.Rect.T) - p.top(rel, w.Rect.W())
				if oy <= surf && fl.y >= surf {
					if covered(f.Windows[:wi], fl.x, surf-0.5) {
						continue // the point is hidden behind a higher window
					}
					p.add(rel, fl.r*1.2*float32(s.cfg.AccumRate), float32(s.cfg.MaxPile))
					landed = true
					break
				}
			}
		}
		if landed || fl.y > H+4 || fl.x < -0.2*W || fl.x > 1.2*W {
			*fl = s.newFlake(false)
		}
	}
	for _, p := range s.piles {
		p.relax()
	}
	live := s.puffs[:0]
	for _, q := range s.puffs {
		q.life -= fdt
		if q.life <= 0 {
			continue
		}
		q.vy += 220 * fdt
		q.vx *= 1 - 1.8*fdt
		q.x += q.vx * fdt
		q.y += q.vy * fdt
		live = append(live, q)
	}
	s.puffs = live
}

// Draw renders everything onto c (which should have been cleared).
func (s *Sim) Draw(c *Canvas) {
	if s.cfg.Accumulate {
		s.drawPiles(c)
	}
	s.drawSnowmen(c)
	for _, q := range s.puffs {
		c.Disc(q.x, q.y, q.r, Color{255, 255, 255}, 0.95*clamp01(q.life/q.maxLife*1.6))
	}
	for _, fl := range s.flakes {
		white := Color{255, 255, 255}
		if fl.r > 2 {
			c.Disc(fl.x, fl.y, fl.r*1.9, white, fl.alpha*0.14)
		}
		c.Disc(fl.x, fl.y, fl.r, white, fl.alpha)
	}
}

func (s *Sim) drawPiles(c *Canvas) {
	wins := s.frame.Windows
	for wi, w := range wins {
		p := s.piles[w.ID]
		if p == nil || !s.eligible(w) {
			continue
		}
		width := w.Rect.W()
		for rel := 0; rel < width; rel++ {
			x := w.Rect.L + rel
			if x < 0 || x >= c.W {
				continue
			}
			h := p.top(rel, width)
			if h < 0.25 {
				continue
			}
			yTop := float32(w.Rect.T) - h
			y0 := int(math.Floor(float64(yTop)))
			segs := [][2]int{{y0, w.Rect.T + 1}}
			for _, o := range wins[:wi] { // clip away whatever is in front of this window
				if x < o.Rect.L || x >= o.Rect.R {
					continue
				}
				var next [][2]int
				for _, sg := range segs {
					if o.Rect.B <= sg[0] || o.Rect.T >= sg[1] {
						next = append(next, sg)
						continue
					}
					if o.Rect.T > sg[0] {
						next = append(next, [2]int{sg[0], o.Rect.T})
					}
					if o.Rect.B < sg[1] {
						next = append(next, [2]int{o.Rect.B, sg[1]})
					}
				}
				segs = next
			}
			for _, sg := range segs {
				for y := sg[0]; y < sg[1]; y++ {
					a := float32(1)
					if y == y0 {
						a = 1 - (yTop - float32(y0))
					}
					depth := clamp01((float32(y) - yTop) / max(h, 1))
					col := Color{
						R: uint8(255 - 38*depth*depth),
						G: uint8(255 - 24*depth*depth),
						B: 255 - uint8(7*depth),
					}
					c.blend(x, y, col, a*0.97)
				}
			}
		}
	}
}
