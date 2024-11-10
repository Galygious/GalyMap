package ui

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

// Define a variable to store the GDI+ token
var gdiplusToken uintptr

// GdipStartup initializes GDI+ for graphical operations.
func GdipStartup() error {
	// Define GDI+ startup input parameters
	type GdiplusStartupInput struct {
		GdiplusVersion           uint32
		DebugEventCallback       uintptr
		SuppressBackgroundThread int32
		SuppressExternalCodecs   int32
	}
	gdiplusStartupInput := GdiplusStartupInput{
		GdiplusVersion: 1,
	}

	// Load GDI+ and call GdiplusStartup to initialize
	gdiplus := syscall.NewLazyDLL("gdiplus.dll")
	gdiplusStartup := gdiplus.NewProc("GdiplusStartup")
	status, _, err := gdiplusStartup.Call(
		uintptr(unsafe.Pointer(&gdiplusToken)),
		uintptr(unsafe.Pointer(&gdiplusStartupInput)),
		0,
	)
	if status != 0 {
		return fmt.Errorf("GDI+ startup failed: %v", err)
	}
	log.Println("GDI+ successfully initialized.")
	return nil
}

// GdipShutdown releases GDI+ resources on program exit.
func GdipShutdown() {
	gdiplus := syscall.NewLazyDLL("gdiplus.dll")
	gdiplusShutdown := gdiplus.NewProc("GdiplusShutdown")
	gdiplusShutdown.Call(gdiplusToken)
	log.Println("GDI+ resources released.")
}
