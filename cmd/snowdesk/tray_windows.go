//go:build windows

package main

import (
	"os"
	"unsafe"

	"snowdesk/snow"
)

const (
	idPause = 1 + iota
	idSnowmen
	idPiles
	idLight
	idNormal
	idHeavy
	idClear
	idAutostart
	idExit
)

const trayID = 1

var trayIcon uintptr

// makeIcon renders the tray icon with the same drawing code the previews use.
func makeIcon() uintptr {
	const sz = 32
	c := snow.NewCanvas(sz, sz)
	snow.DrawIcon(c)

	scratchBI = bitmapInfo{Header: bitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: sz, Height: -sz, Planes: 1, BitCount: 32,
	}}
	color, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&scratchBI)), 0, uintptr(unsafe.Pointer(&dibBits)), 0, 0)
	if color == 0 {
		return 0
	}
	dst := unsafe.Slice((*uint32)(dibBits), sz*sz)
	for i, p := range c.Pix { // icons want straight alpha, the canvas is premultiplied
		a := p >> 24
		if a == 0 {
			dst[i] = 0
			continue
		}
		un := func(v uint32) uint32 { return min(255, v*255/a) }
		dst[i] = a<<24 | un(p>>16&0xff)<<16 | un(p>>8&0xff)<<8 | un(p&0xff)
	}
	mask, _, _ := pCreateBitmap.Call(sz, sz, 1, 1, uintptr(unsafe.Pointer(&scratchMask[0])))
	scratchIcon = iconInfo{FIcon: 1, MaskBmp: mask, ColorBmp: color}
	h, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&scratchIcon)))
	pDeleteObject.Call(color)
	pDeleteObject.Call(mask)
	return h
}

func fillNID(hwnd uintptr) {
	scratchNID = notifyIconData{Hwnd: hwnd, ID: trayID, Flags: nifMsg | nifIcon | nifTip, CallbackMessage: wmTray, Icon: trayIcon}
	scratchNID.Size = uint32(unsafe.Sizeof(scratchNID))
	copy(scratchNID.Tip[:], utf16Slice("SnowDesk"))
}

func utf16Slice(s string) []uint16 {
	u := make([]uint16, 0, len(s)+1)
	for _, r := range s {
		u = append(u, uint16(r))
	}
	return append(u, 0)
}

func addTrayIcon(hwnd uintptr) {
	if trayIcon == 0 {
		trayIcon = makeIcon()
	}
	fillNID(hwnd)
	pShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&scratchNID)))
}

func removeTrayIcon(hwnd uintptr) {
	fillNID(hwnd)
	pShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&scratchNID)))
	if trayIcon != 0 {
		pDestroyIcon.Call(trayIcon)
		trayIcon = 0
	}
}

func (a *app) showMenu(hwnd uintptr) {
	menu, _, _ := pCreatePopupMenu.Call()
	defer pDestroyMenu.Call(menu)
	add := func(id int, text string, checked bool) {
		flags := uintptr(mfString)
		if checked {
			flags |= mfChecked
		}
		pAppendMenuW.Call(menu, flags, uintptr(id), uintptr(unsafe.Pointer(utf16(text))))
	}
	sep := func() { pAppendMenuW.Call(menu, mfSeparator, 0, 0) }

	pauseText := "Pause snow"
	if a.paused {
		pauseText = "Resume snow"
	}
	add(idPause, pauseText, false)
	sep()
	add(idSnowmen, "Snowmen", a.cfg.Snowmen > 0)
	add(idPiles, "Snow on windows", a.cfg.Accumulate)
	sep()
	add(idLight, "Light snowfall", a.cfg.Density <= 80)
	add(idNormal, "Normal snowfall", a.cfg.Density > 80 && a.cfg.Density < 200)
	add(idHeavy, "Blizzard", a.cfg.Density >= 200)
	sep()
	add(idClear, "Clear snow from windows", false)
	add(idAutostart, "Start with Windows", autostartEnabled())
	sep()
	add(idExit, "Exit", false)

	pGetCursorPos.Call(uintptr(unsafe.Pointer(&scratchPoint)))
	x, y := scratchPoint.X, scratchPoint.Y
	pSetForegroundWindow.Call(hwnd) // required, or the menu never closes when clicking elsewhere
	cmd, _, _ := pTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmBottomAlign, uintptr(x), uintptr(y), 0, hwnd, 0)
	pPostMessageW.Call(hwnd, wmNull, 0, 0)

	switch int(cmd) {
	case idPause:
		a.paused = !a.paused
	case idSnowmen:
		if a.cfg.Snowmen > 0 {
			a.cfg.Snowmen = 0
		} else {
			a.cfg.Snowmen = 3
		}
	case idPiles:
		a.cfg.Accumulate = !a.cfg.Accumulate
	case idLight:
		a.cfg.Density = 60
	case idNormal:
		a.cfg.Density = 120
	case idHeavy:
		a.cfg.Density = 260
	case idClear:
		a.sim.ClearPiles()
	case idAutostart:
		setAutostart(!autostartEnabled())
	case idExit:
		pDestroyWindow.Call(hwnd)
		return
	default:
		return
	}
	a.sim.SetConfig(a.cfg)
	_ = snow.SaveConfig(a.cfg)
}

// --- "Start with Windows" via the per-user Run key (no admin rights needed) ---

const (
	hkeyCurrentUser = 0x80000001
	runKeyPath      = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName    = "SnowDesk"
	keySetValue     = 0x0002
	keyQueryValue   = 0x0001
	regSZ           = 1
)

func openRunKey() uintptr {
	var hk uintptr
	r, _, _ := pRegCreateKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(utf16(runKeyPath))), 0, 0, 0,
		keySetValue|keyQueryValue, 0, uintptr(unsafe.Pointer(&hk)), 0)
	if r != 0 {
		return 0
	}
	return hk
}

func autostartEnabled() bool {
	hk := openRunKey()
	if hk == 0 {
		return false
	}
	defer pRegCloseKey.Call(hk)
	r, _, _ := pRegQueryValueEx.Call(hk, uintptr(unsafe.Pointer(utf16(runValueName))), 0, 0, 0, 0)
	return r == 0
}

func setAutostart(on bool) {
	hk := openRunKey()
	if hk == 0 {
		return
	}
	defer pRegCloseKey.Call(hk)
	name := utf16(runValueName)
	if !on {
		pRegDeleteValueW.Call(hk, uintptr(unsafe.Pointer(name)))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	val := utf16(`"` + exe + `"`)
	n := 0
	for p := unsafe.Pointer(val); *(*uint16)(p) != 0; p = unsafe.Add(p, 2) {
		n++
	}
	pRegSetValueExW.Call(hk, uintptr(unsafe.Pointer(name)), 0, regSZ, uintptr(unsafe.Pointer(val)), uintptr((n+1)*2))
}
