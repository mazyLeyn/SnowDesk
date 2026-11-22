package snow

import "math"

// Color is a straight (non-premultiplied) RGB color.
type Color struct{ R, G, B uint8 }

// Canvas is a premultiplied-alpha 0xAARRGGBB pixel buffer. That is exactly the
// layout Windows expects for a per-pixel-alpha layered window, so on Windows
// Pix points straight at the DIB section memory.
type Canvas struct {
	W, H int
	Pix  []uint32
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pix: make([]uint32, w*h)}
}

func (c *Canvas) Clear() { clear(c.Pix) }

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func u8(v float32) uint32 {
	if v >= 255 {
		return 255
	}
	if v <= 0 {
		return 0
	}
	return uint32(v + 0.5)
}

// blend composites col with opacity a over pixel (x, y).
func (c *Canvas) blend(x, y int, col Color, a float32) {
	if a <= 0 || x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	if a > 1 {
		a = 1
	}
	i := y*c.W + x
	d := c.Pix[i]
	inv := 1 - a
	na := a*255 + float32(d>>24)*inv
	nr := float32(col.R)*a + float32(d>>16&0xff)*inv
	ng := float32(col.G)*a + float32(d>>8&0xff)*inv
	nb := float32(col.B)*a + float32(d&0xff)*inv
	c.Pix[i] = u8(na)<<24 | u8(nr)<<16 | u8(ng)<<8 | u8(nb)
}

func (c *Canvas) bounds(x0, y0, x1, y1 float32) (ix0, iy0, ix1, iy1 int) {
	ix0 = max(int(math.Floor(float64(x0))), 0)
	iy0 = max(int(math.Floor(float64(y0))), 0)
	ix1 = min(int(math.Ceil(float64(x1))), c.W-1)
	iy1 = min(int(math.Ceil(float64(y1))), c.H-1)
	return
}

// Disc draws an anti-aliased filled circle.
func (c *Canvas) Disc(cx, cy, r float32, col Color, alpha float32) {
	x0, y0, x1, y1 := c.bounds(cx-r-1, cy-r-1, cx+r+1, cy+r+1)
	for y := y0; y <= y1; y++ {
		dy := float32(y) + 0.5 - cy
		for x := x0; x <= x1; x++ {
			dx := float32(x) + 0.5 - cx
			d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if cov := clamp01(r - d + 0.5); cov > 0 {
				c.blend(x, y, col, cov*alpha)
			}
		}
	}
}

// Ellipse draws an anti-aliased filled axis-aligned ellipse.
func (c *Canvas) Ellipse(cx, cy, rx, ry float32, col Color, alpha float32) {
	x0, y0, x1, y1 := c.bounds(cx-rx-1, cy-ry-1, cx+rx+1, cy+ry+1)
	m := min(rx, ry)
	for y := y0; y <= y1; y++ {
		dy := (float32(y) + 0.5 - cy) / ry
		for x := x0; x <= x1; x++ {
			dx := (float32(x) + 0.5 - cx) / rx
			d := (float32(math.Sqrt(float64(dx*dx+dy*dy))) - 1) * m
			if cov := clamp01(0.5 - d); cov > 0 {
				c.blend(x, y, col, cov*alpha)
			}
		}
	}
}

// Line draws an anti-aliased line segment with round caps and width w.
func (c *Canvas) Line(x0, y0, x1, y1, w float32, col Color, alpha float32) {
	hw := w / 2
	bx0, by0, bx1, by1 := c.bounds(min(x0, x1)-hw-1, min(y0, y1)-hw-1, max(x0, x1)+hw+1, max(y0, y1)+hw+1)
	vx, vy := x1-x0, y1-y0
	l2 := vx*vx + vy*vy
	for y := by0; y <= by1; y++ {
		py := float32(y) + 0.5
		for x := bx0; x <= bx1; x++ {
			px := float32(x) + 0.5
			var t float32
			if l2 > 0 {
				t = clamp01(((px-x0)*vx + (py-y0)*vy) / l2)
			}
			dx, dy := px-(x0+t*vx), py-(y0+t*vy)
			d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if cov := clamp01(hw - d + 0.5); cov > 0 {
				c.blend(x, y, col, cov*alpha)
			}
		}
	}
}

// FillRect draws an axis-aligned rectangle with anti-aliased (fractional) edges.
func (c *Canvas) FillRect(x0, y0, x1, y1 float32, col Color, alpha float32) {
	bx0, by0, bx1, by1 := c.bounds(x0, y0, x1, y1)
	for y := by0; y <= by1; y++ {
		cy := clamp01(min(y1, float32(y+1)) - max(y0, float32(y)))
		if cy <= 0 {
			continue
		}
		for x := bx0; x <= bx1; x++ {
			cx := clamp01(min(x1, float32(x+1)) - max(x0, float32(x)))
			if cx > 0 {
				c.blend(x, y, col, cx*cy*alpha)
			}
		}
	}
}

// DrawIcon paints the tray icon (a white snowflake on a blue disc) onto c.
func DrawIcon(c *Canvas) {
	s := float32(c.W)
	mid := s / 2
	c.Disc(mid, mid, mid-0.5, Color{38, 99, 200}, 1)
	c.Disc(mid, mid*0.9, mid*0.86, Color{58, 128, 232}, 0.55)
	white := Color{255, 255, 255}
	arm := s * 0.34
	for i := 0; i < 6; i++ {
		a := float64(i) * math.Pi / 3
		dx, dy := float32(math.Cos(a)), float32(math.Sin(a))
		c.Line(mid, mid, mid+dx*arm, mid+dy*arm, s*0.075, white, 1)
		bx, by := mid+dx*arm*0.62, mid+dy*arm*0.62
		for _, off := range []float64{-0.75, 0.75} {
			ba := a + off
			c.Line(bx, by, bx+float32(math.Cos(ba))*arm*0.34, by+float32(math.Sin(ba))*arm*0.34, s*0.055, white, 1)
		}
	}
	c.Disc(mid, mid, s*0.06, white, 1)
}
