//go:build windows

package main

import (
	"unsafe"

	"snowdesk/snow"
)

// overlay is one borderless, click-through, always-on-top window with
// per-pixel alpha. The simulation draws straight into its DIB section.
type overlay struct {
	hwnd   uintptr
	memDC  uintptr
	bmp    uintptr
	oldBmp uintptr
	X, Y   int
	W, H   int
	canvas *snow.Canvas
}

const className_ = "SnowDeskOverlay"

func registerClass(wndProc uintptr) uintptr {
	inst, _, _ := pGetModuleHandleW.Call(0)
	wc := wndClassEx{
		WndProc:   wndProc,
		Instance:  inst,
		ClassName: utf16(className_),
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return inst
}

func newOverlay(wndProc uintptr, x, y, w, h int) *overlay {
	inst := registerClass(wndProc)
	o := &overlay{X: x, Y: y, W: w, H: h}
	const ex = wsExLayered | wsExTransparent | wsExTopmost | wsExToolWindow | wsExNoActivate
	o.hwnd, _, _ = pCreateWindowExW.Call(
		ex,
		uintptr(unsafe.Pointer(utf16(className_))),
		uintptr(unsafe.Pointer(utf16("SnowDesk"))),
		wsPopup,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0, 0, inst, 0)
	o.memDC, _, _ = pCreateCompatibleDC.Call(0)
	o.makeDIB()
	return o
}

func (o *overlay) makeDIB() {
	if o.bmp != 0 {
		pSelectObject.Call(o.memDC, o.oldBmp)
		pDeleteObject.Call(o.bmp)
	}
	scratchBI = bitmapInfo{Header: bitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(o.W), Height: -int32(o.H), Planes: 1, BitCount: 32,
	}}
	o.bmp, _, _ = pCreateDIBSection.Call(o.memDC, uintptr(unsafe.Pointer(&scratchBI)), 0,
		uintptr(unsafe.Pointer(&dibBits)), 0, 0)
	o.oldBmp, _, _ = pSelectObject.Call(o.memDC, o.bmp)
	o.canvas = &snow.Canvas{W: o.W, H: o.H, Pix: unsafe.Slice((*uint32)(dibBits), o.W*o.H)}
}

func (o *overlay) resize(x, y, w, h int) {
	o.X, o.Y, o.W, o.H = x, y, w, h
	pSetWindowPos.Call(o.hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder|swpNoActivate)
	o.makeDIB()
}

func (o *overlay) show(visible bool) {
	if visible {
		pShowWindow.Call(o.hwnd, swShowNoActivate)
	} else {
		pShowWindow.Call(o.hwnd, swHide)
	}
}

// raise re-asserts topmost status; the taskbar likes to steal it.
func (o *overlay) raise() {
	const hwndTopmost = ^uintptr(0) // (HWND)-1
	pSetWindowPos.Call(o.hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
}

// present pushes the canvas to the screen.
func (o *overlay) present() {
	scratchSize = size{int32(o.W), int32(o.H)}
	scratchPoint = point{}
	scratchBlend = blendFunction{BlendOp: 0 /* AC_SRC_OVER */, SourceConstantAlpha: 255, AlphaFormat: 1 /* AC_SRC_ALPHA */}
	pUpdateLayeredWindow.Call(o.hwnd, 0, 0,
		uintptr(unsafe.Pointer(&scratchSize)), o.memDC,
		uintptr(unsafe.Pointer(&scratchPoint)), 0,
		uintptr(unsafe.Pointer(&scratchBlend)), ulwAlpha)
}

func (o *overlay) rect() snow.Rect { return snow.Rect{L: o.X, T: o.Y, R: o.X + o.W, B: o.Y + o.H} }
