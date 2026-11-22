//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Only the standard library is used: every Win32 function is called through
// syscall.LazyDLL, loaded from System32 explicitly so a DLL planted next to the
// exe can never be picked up.
func sysDLL(name string) *syscall.LazyDLL {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return syscall.NewLazyDLL(filepath.Join(root, "System32", name))
}

var (
	user32   = sysDLL("user32.dll")
	gdi32    = sysDLL("gdi32.dll")
	kernel32 = sysDLL("kernel32.dll")
	shell32  = sysDLL("shell32.dll")
	dwmapi   = sysDLL("dwmapi.dll")
	advapi32 = sysDLL("advapi32.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pDestroyWindow            = user32.NewProc("DestroyWindow")
	pShowWindow               = user32.NewProc("ShowWindow")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pUpdateLayeredWindow      = user32.NewProc("UpdateLayeredWindow")
	pPeekMessageW             = user32.NewProc("PeekMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pMsgWaitForMultipleObjEx  = user32.NewProc("MsgWaitForMultipleObjectsEx")
	pGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	pEnumWindows              = user32.NewProc("EnumWindows")
	pIsWindowVisible          = user32.NewProc("IsWindowVisible")
	pIsIconic                 = user32.NewProc("IsIconic")
	pGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	pGetClassNameW            = user32.NewProc("GetClassNameW")
	pGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	pGetWindowRect            = user32.NewProc("GetWindowRect")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	pAppendMenuW              = user32.NewProc("AppendMenuW")
	pTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	pDestroyMenu              = user32.NewProc("DestroyMenu")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	pSetProcessDpiAwarenessCx = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
	pSystemParametersInfoW    = user32.NewProc("SystemParametersInfoW")
	pCreateIconIndirect       = user32.NewProc("CreateIconIndirect")
	pDestroyIcon              = user32.NewProc("DestroyIcon")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW     = kernel32.NewProc("CreateMutexW")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")

	pDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")

	pRegCreateKeyExW = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW  = advapi32.NewProc("RegSetValueExW")
	pRegQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	pRegDeleteValueW = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey     = advapi32.NewProc("RegCloseKey")
)

const (
	wsPopup = 0x80000000

	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000

	gwlStyle   = -16
	gwlExStyle = -20
	wsCaption  = 0x00C00000

	swHide           = 0
	swShowNoActivate = 4

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	ulwAlpha = 2

	wmDestroy       = 0x0002
	wmQuit          = 0x0012
	wmDisplayChange = 0x007E
	wmNull          = 0x0000
	wmLButtonUp     = 0x0202
	wmRButtonUp     = 0x0205
	wmTray          = 0x8001 // WM_APP + 1

	pmRemove   = 0x0001
	qsAllInput = 0x04FF

	mfString    = 0x0000
	mfChecked   = 0x0008
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
	tpmReturnCmd   = 0x0100

	nimAdd    = 0
	nimDelete = 2
	nifMsg    = 0x1
	nifIcon   = 0x2
	nifTip    = 0x4

	smCxScreen         = 0
	smCyScreen         = 1
	smXVirtualScreen   = 76
	smYVirtualScreen   = 77
	smCxVirtualScreen  = 78
	smCyVirtualScreen  = 79
	spiGetWorkArea     = 0x0030
	dwmwaCloaked       = 14
	dwmwaExtFrameBound = 9
)

type rect struct{ Left, Top, Right, Bottom int32 }
type point struct{ X, Y int32 }
type size struct{ CX, CY int32 }

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

type blendFunction struct {
	BlendOp, BlendFlags, SourceConstantAlpha, AlphaFormat byte
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	MaskBmp  uintptr
	ColorBmp uintptr
}

type notifyIconData struct {
	Size             uint32
	Hwnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         [4]uint32
	BalloonIcon      uintptr
}

// Scratch structures live at package level so their addresses never move while
// a Win32 call is using them (everything runs on one locked OS thread).
var (
	scratchMsg   msg
	scratchRect  rect
	scratchPoint point
	scratchSize  size
	scratchBI    bitmapInfo
	scratchBlend blendFunction
	scratchNID   notifyIconData
	scratchIcon  iconInfo
	scratchClass [256]uint16
	scratchMask  [32 * 4]byte
	dibBits      unsafe.Pointer
)

func utf16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func className(h uintptr) string {
	n, _, _ := pGetClassNameW.Call(h, uintptr(unsafe.Pointer(&scratchClass[0])), uintptr(len(scratchClass)))
	return syscall.UTF16ToString(scratchClass[:n])
}

func getLong(h uintptr, index int) uintptr {
	r, _, _ := pGetWindowLongPtrW.Call(h, uintptr(index))
	return r
}

func metric(i int) int {
	r, _, _ := pGetSystemMetrics.Call(uintptr(i))
	return int(int32(r))
}
