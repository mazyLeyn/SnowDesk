//go:build windows

// SnowDesk puts falling snow, snowmen and snow-covered window title bars on
// top of your Windows desktop. Everything is drawn into a transparent,
// click-through overlay, so it never gets in the way of your work.
package main

import (
	"flag"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"snowdesk/snow"
)

type app struct {
	cfg            snow.Config
	sim            *snow.Sim
	ov             *overlay
	paused         bool
	shown          bool
	displayChanged bool
	taskbarMsg     uint32
}

var a app

func init() {
	// Win32 windows belong to the thread that created them: pin the main goroutine.
	runtime.LockOSThread()
}

var wndProc = syscall.NewCallback(func(hwnd, m, wp, lp uintptr) uintptr {
	switch uint32(m) {
	case wmTray:
		if uint32(lp) == wmRButtonUp || uint32(lp) == wmLButtonUp {
			a.showMenu(hwnd)
		}
		return 0
	case wmDisplayChange:
		a.displayChanged = true
		return 0
	case wmDestroy:
		removeTrayIcon(hwnd)
		pPostQuitMessage.Call(0)
		return 0
	}
	if a.taskbarMsg != 0 && uint32(m) == a.taskbarMsg { // Explorer restarted: put the icon back
		addTrayIcon(hwnd)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, m, wp, lp)
	return r
})

func singleInstance() bool {
	_, _, err := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16(`Local\SnowDeskSingleInstance`))))
	return err != syscall.Errno(183) // ERROR_ALREADY_EXISTS
}

func setDPIAware() {
	v := -4 // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	if r, _, _ := pSetProcessDpiAwarenessCx.Call(uintptr(v)); r == 0 {
		pSetProcessDPIAware.Call()
	}
}

func screenRect(all bool) (x, y, w, h int) {
	if all {
		return metric(smXVirtualScreen), metric(smYVirtualScreen), metric(smCxVirtualScreen), metric(smCyVirtualScreen)
	}
	return 0, 0, metric(smCxScreen), metric(smCyScreen)
}

func parseFlags() snow.Config {
	cfg := snow.LoadConfig()
	flag.Float64Var(&cfg.Density, "density", cfg.Density, "snowflakes per megapixel (60 = light, 120 = normal, 260 = blizzard)")
	flag.Float64Var(&cfg.Wind, "wind", cfg.Wind, "wind strength in px/s")
	flag.IntVar(&cfg.Snowmen, "snowmen", cfg.Snowmen, "number of snowmen (0 = none)")
	flag.Float64Var(&cfg.SnowmanScale, "snowman-scale", cfg.SnowmanScale, "snowman size multiplier")
	flag.BoolVar(&cfg.Accumulate, "piles", cfg.Accumulate, "let snow pile up on window title bars")
	flag.Float64Var(&cfg.AccumRate, "pile-rate", cfg.AccumRate, "how fast snow piles up")
	flag.Float64Var(&cfg.MaxPile, "max-pile", cfg.MaxPile, "maximum pile height in px")
	flag.IntVar(&cfg.FPS, "fps", cfg.FPS, "frames per second")
	flag.BoolVar(&cfg.AllMonitors, "all-monitors", cfg.AllMonitors, "cover every monitor, not just the primary one")
	flag.Parse()
	cfg.Sanitize()
	return cfg
}

func wait(d time.Duration) {
	ms := max(int(d/time.Millisecond), 1)
	pMsgWaitForMultipleObjEx.Call(0, 0, uintptr(ms), qsAllInput, 4 /* MWMO_INPUTAVAILABLE */)
}

// pump handles pending window messages and reports whether the app should quit.
func pump() bool {
	for {
		if r, _, _ := pPeekMessageW.Call(uintptr(unsafe.Pointer(&scratchMsg)), 0, 0, 0, pmRemove); r == 0 {
			return false
		}
		if scratchMsg.Message == wmQuit {
			return true
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&scratchMsg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&scratchMsg)))
	}
}

func main() {
	if !singleInstance() {
		return
	}
	a.cfg = parseFlags()
	setDPIAware()

	x, y, w, h := screenRect(a.cfg.AllMonitors)
	a.ov = newOverlay(wndProc, x, y, w, h)
	if a.ov.hwnd == 0 {
		return
	}
	tr.self, tr.ox, tr.oy = a.ov.hwnd, x, y
	a.sim = snow.New(w, h, a.cfg, time.Now().UnixNano())
	a.taskbarMsg = func() uint32 {
		r, _, _ := pRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16("TaskbarCreated"))))
		return uint32(r)
	}()
	addTrayIcon(a.ov.hwnd)

	var (
		wins     []snow.Window
		frame    snow.Frame
		last     = time.Now()
		lastScan time.Time
		lastTop  time.Time
		hidden   bool // a fullscreen app is in front
	)

	for {
		if pump() {
			return
		}
		if a.displayChanged {
			a.displayChanged = false
			x, y, w, h = screenRect(a.cfg.AllMonitors)
			a.ov.resize(x, y, w, h)
			tr.ox, tr.oy = x, y
			a.sim.Resize(w, h)
		}

		visible := !a.paused && !hidden
		if visible != a.shown {
			a.ov.show(visible)
			a.shown = visible
		}
		if a.paused {
			wait(150 * time.Millisecond)
			last = time.Now()
			continue
		}

		now := time.Now()
		if now.Sub(lastScan) >= 120*time.Millisecond {
			lastScan = now
			hidden = fullscreenAppActive(a.ov.rect(), a.ov.hwnd)
			if !hidden {
				wins = tr.snapshot()
				wl, _, wr, wb := workArea()
				frame = snow.Frame{Windows: wins, Ground: wb - y, StandL: wl - x, StandR: wr - x}
			}
		}
		if now.Sub(lastTop) >= 2*time.Second {
			lastTop = now
			a.ov.raise()
		}
		if hidden {
			wait(150 * time.Millisecond)
			last = time.Now()
			continue
		}

		a.sim.Update(now.Sub(last).Seconds(), frame)
		last = now
		a.ov.canvas.Clear()
		a.sim.Draw(a.ov.canvas)
		a.ov.present()

		if rest := time.Second/time.Duration(a.cfg.FPS) - time.Since(now); rest > 0 {
			wait(rest)
		}
	}
}
