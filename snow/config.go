package snow

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds every user-tunable setting. It is stored as JSON in
// %AppData%\SnowDesk\config.json.
type Config struct {
	Density      float64 `json:"density"`          // snowflakes per megapixel of screen
	Wind         float64 `json:"wind"`             // wind strength in px/s
	Snowmen      int     `json:"snowmen"`          // number of snowmen standing on the taskbar
	SnowmanScale float64 `json:"snowmanScale"`     // 1.0 = ~60px tall
	Accumulate   bool    `json:"accumulate"`       // snow piles up on window title bars
	AccumRate    float64 `json:"accumulationRate"` // 1.0 = full pile in a few minutes
	MaxPile      float64 `json:"maxPile"`          // max pile height in px
	FPS          int     `json:"fps"`
	AllMonitors  bool    `json:"allMonitors"` // cover the whole virtual desktop, not just the primary monitor
}

func DefaultConfig() Config {
	return Config{
		Density:      120,
		Wind:         18,
		Snowmen:      0,
		SnowmanScale: 1.3,
		Accumulate:   true,
		AccumRate:    1,
		MaxPile:      10,
		FPS:          30,
	}
}

// Sanitize clamps every value into a sane range so a hand-edited config file
// can never crash or freeze the overlay.
func (c *Config) Sanitize() {
	c.Density = clampF(c.Density, 0, 1500)
	c.Wind = clampF(c.Wind, -200, 200)
	c.Snowmen = int(clampF(float64(c.Snowmen), 0, 20))
	c.SnowmanScale = clampF(c.SnowmanScale, 0.3, 4)
	c.AccumRate = clampF(c.AccumRate, 0, 20)
	c.MaxPile = clampF(c.MaxPile, 0, 40)
	c.FPS = int(clampF(float64(c.FPS), 10, 120))
}

func clampF(v, lo, hi float64) float64 {
	if v != v { // NaN
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ConfigPath returns the location of the config file.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SnowDesk", "config.json"), nil
}

// LoadConfig reads the config file, falling back to defaults for anything
// missing or unreadable.
func LoadConfig() Config {
	cfg := DefaultConfig()
	if p, err := ConfigPath(); err == nil {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &cfg)
		}
	}
	cfg.Sanitize()
	return cfg
}

// SaveConfig writes the config file, creating the folder if needed.
func SaveConfig(cfg Config) error {
	p, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
