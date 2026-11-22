package snow

import "math"

type snowman struct {
	fx    float32 // 0..1 position inside the stand range
	scale float32
	scarf Color
	phase float32
}

var scarfColors = []Color{{205, 45, 55}, {40, 140, 80}, {50, 110, 200}, {230, 170, 40}}

func (s *Sim) newSnowmen(n int) []snowman {
	men := make([]snowman, n)
	for i := range men {
		men[i] = snowman{
			fx:    (float32(i) + 0.5 + (s.rng.Float32()-0.5)*0.6) / float32(n),
			scale: float32(s.cfg.SnowmanScale) * (0.85 + s.rng.Float32()*0.35),
			scarf: scarfColors[s.rng.Intn(len(scarfColors))],
			phase: s.rng.Float32() * 6.28,
		}
	}
	return men
}

func (s *Sim) drawSnowmen(c *Canvas) {
	ground := s.frame.Ground
	if ground <= 0 || ground > s.H {
		ground = s.H
	}
	l, r := s.frame.StandL, s.frame.StandR
	if l == 0 && r == 0 {
		r = s.W
	}
	for _, m := range s.men {
		x := float32(l) + m.fx*float32(r-l)
		s.drawSnowman(c, x, float32(ground)+2, m)
	}
}

var (
	coal   = Color{36, 36, 44}
	twig   = Color{98, 64, 40}
	carrot = Color{242, 128, 32}
)

func (s *Sim) drawSnowman(c *Canvas, x, g float32, m snowman) {
	u := m.scale
	t := float32(s.t)
	r1, r2, r3 := 15*u, 11.5*u, 8.5*u
	y1 := g - r1
	y2 := y1 - (r1+r2)*0.8
	y3 := y2 - (r2+r3)*0.8

	c.Ellipse(x+3*u, g, r1*1.15, 2.2*u, Color{15, 25, 60}, 0.28) // soft shadow

	// arms (behind the body), gently waving
	for _, side := range []float32{-1, 1} {
		a := 0.55 + 0.13*float32(math.Sin(float64(t*1.4+m.phase+side)))
		sx, sy := x+side*r2*0.82, y2-r2*0.1
		ex, ey := sx+side*float32(math.Cos(float64(a)))*19*u, sy-float32(math.Sin(float64(a)))*19*u
		c.Line(sx, sy, ex, ey, 1.9*u, twig, 1)
		fx, fy := sx+(ex-sx)*0.62, sy+(ey-sy)*0.62
		c.Line(fx, fy, fx+side*5*u, fy-5.5*u, 1.4*u, twig, 1)
	}

	ball := func(cx, cy, r float32) {
		c.Disc(cx, cy, r+0.9, Color{120, 150, 200}, 0.6)
		c.Disc(cx, cy, r, Color{222, 233, 250}, 1)
		c.Disc(cx-r*0.2, cy-r*0.22, r*0.8, Color{244, 248, 255}, 0.95)
		c.Disc(cx-r*0.3, cy-r*0.36, r*0.42, Color{255, 255, 255}, 0.85)
	}
	ball(x, y1, r1)
	ball(x, y2, r2)

	for i := 0; i < 3; i++ { // coal buttons
		c.Disc(x, y2+float32(i-1)*5*u+u, 1.3*u, coal, 1)
	}

	ball(x, y3, r3)

	// scarf with a fluttering tail
	ny := y3 + r3*0.8
	c.Ellipse(x, ny, r3*0.98, 2.7*u, m.scarf, 1)
	flutter := clampF32(s.wind*0.12, -4*u, 4*u) + 1.8*u*float32(math.Sin(float64(t*2.2+m.phase)))
	tx := x + r3*0.55
	c.Line(tx, ny+1*u, tx+flutter*0.5, ny+6*u, 3.2*u, m.scarf, 1)
	c.Line(tx+flutter*0.5, ny+6*u, tx+flutter, ny+11*u, 3.2*u, m.scarf, 1)

	// face
	for _, side := range []float32{-1, 1} {
		c.Disc(x+side*2.8*u, y3-1.8*u, 1.15*u, coal, 1)
	}
	for i := 0; i < 4; i++ { // carrot nose
		k := float32(i)
		c.Disc(x+1.2*u+k*1.9*u, y3+0.6*u+k*0.28*u, (1.7-0.32*k)*u, carrot, 1)
	}
	for k := -2; k <= 2; k++ { // smile
		fk := float32(k)
		c.Disc(x+fk*1.7*u, y3+5*u-0.3*u*fk*fk, 0.65*u, coal, 1)
	}

	// top hat
	brimY := y3 - r3*0.8
	hatTop := brimY - 11*u
	c.Ellipse(x, brimY, r3*1.2, 2.2*u, Color{24, 24, 30}, 1)
	c.FillRect(x-5.3*u, hatTop, x+5.3*u, brimY, Color{30, 30, 38}, 1)
	c.FillRect(x-5.3*u, brimY-2.6*u, x+5.3*u, brimY, Color{190, 40, 52}, 1)
	c.Ellipse(x, hatTop, 5.3*u, 1.5*u, Color{44, 44, 54}, 1)
	c.Ellipse(x, hatTop-0.4*u, 4.6*u, 1.1*u, Color{248, 250, 255}, 0.95) // snow on the hat
}

func clampF32(v, lo, hi float32) float32 { return max(lo, min(v, hi)) }
