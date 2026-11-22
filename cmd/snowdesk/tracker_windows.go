//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"snowdesk/snow"
)

// tracker lists the visible top-level windows (topmost first) so snow can rest
// on their title bars.
type tracker struct {
	self   uintptr
	ox, oy int // overlay origin on the virtual desktop
	wins   []snow.Window
}

var tr tracker

// Created once: Go only allows a limited number of syscall callbacks.
var enumCallback = syscall.NewCallback(func(h, _ uintptr) uintptr {
	tr.visit(h)
	return 1
})

func (t *tracker) snapshot() []snow.Window {
	t.wins = t.wins[:0]
	pEnumWindows.Call(enumCallback, 0)
	return t.wins
}

func isTaskbar(cls string) bool { return cls == "Shell_TrayWnd" || cls == "Shell_SecondaryTrayWnd" }

func (t *tracker) visit(h uintptr) {
	if h == t.self {
		return
	}
	if r, _, _ := pIsWindowVisible.Call(h); r == 0 {
		return
	}
	if r, _, _ := pIsIconic.Call(h); r != 0 {
		return
	}
	var cloaked uint32 // windows hidden by virtual desktops / UWP suspension
	pDwmGetWindowAttribute.Call(h, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), 4)
	if cloaked != 0 {
		return
	}

	cls := className(h)
	if !isTaskbar(cls) {
		if getLong(h, gwlExStyle)&(wsExToolWindow|wsExTransparent|wsExNoActivate) != 0 {
			return
		}
		switch cls {
		case "Progman", "WorkerW", "Windows.UI.Core.CoreWindow", "ForegroundStaging", "XamlExplorerHostIslandWindow":
			return
		}
		if n, _, _ := pGetWindowTextLengthW.Call(h); n == 0 {
			return
		}
	}

	// The extended frame bounds exclude the invisible resize border Windows 10/11 adds.
	hr, _, _ := pDwmGetWindowAttribute.Call(h, dwmwaExtFrameBound, uintptr(unsafe.Pointer(&scratchRect)), unsafe.Sizeof(scratchRect))
	if hr != 0 {
		if r, _, _ := pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&scratchRect))); r == 0 {
			return
		}
	}
	r := snow.Rect{
		L: int(scratchRect.Left) - t.ox, T: int(scratchRect.Top) - t.oy,
		R: int(scratchRect.Right) - t.ox, B: int(scratchRect.Bottom) - t.oy,
	}
	if r.W() < 80 || r.H() < 30 {
		return
	}
	t.wins = append(t.wins, snow.Window{ID: h, Rect: r})
}

// fullscreenAppActive reports whether a caption-less window (a game, a video in
// fullscreen) covers the whole overlay; we hide so we never sit on top of it.
func fullscreenAppActive(ov snow.Rect, self uintptr) bool {
	h, _, _ := pGetForegroundWindow.Call()
	if h == 0 || h == self {
		return false
	}
	switch className(h) {
	case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd":
		return false
	}
	if getLong(h, gwlStyle)&wsCaption == wsCaption {
		return false
	}
	if r, _, _ := pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&scratchRect))); r == 0 {
		return false
	}
	return int(scratchRect.Left) <= ov.L && int(scratchRect.Top) <= ov.T &&
		int(scratchRect.Right) >= ov.R && int(scratchRect.Bottom) >= ov.B
}

// workArea returns the primary monitor's work area (screen minus taskbar).
func workArea() (l, t, r, b int) {
	pSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&scratchRect)), 0)
	return int(scratchRect.Left), int(scratchRect.Top), int(scratchRect.Right), int(scratchRect.Bottom)
}
