// main.go
package main

import (
	"GalyMap/config"
	"GalyMap/globals"
	"GalyMap/memory"
	"GalyMap/ui"
	"GalyMap/utils"
	"log"
	"syscall"
	"time"

	"github.com/lxn/win"
)

func main() {
	// Initialize log file
	utils.InitializeAppLog()
	log.Println("Application started.")

	// Load configuration
	cfg, err := config.LoadConfig("settings.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded: %+v\n", cfg)

	// Initialize GDI+
	log.Println("Initializing GDI+...")
	err = ui.GdipStartup()
	if err != nil {
		log.Fatalf("GDI+ initialization failed: %v", err)
	}
	defer ui.GdipShutdown()
	log.Println("GDI+ successfully initialized.")

	// Retrieve the HINSTANCE
	hInstance := win.GetModuleHandle(nil)
	if hInstance == 0 {
		log.Fatalf("Failed to get module handle: %v", syscall.GetLastError())
	}

	// Show the process selection window
	selectedProcess, err := ui.ShowProcessSelectionWindow(hInstance)
	if err != nil {
		log.Fatalf("Process selection failed: %v", err)
	}
	if selectedProcess == nil {
		log.Println("No process selected, exiting.")
		return
	}

	// Initialize memory with selected process
	d2r, err := utils.NewClassMemory(selectedProcess.ExeName, 0)
	if err != nil {
		log.Fatalf("Failed to initialize memory: %v", err)
	}
	defer d2r.Close()

	err = memory.PatternScan(d2r)
	if err != nil {
		log.Fatalf("Pattern scan failed: %v", err)
	}

	// Check if player is in-game and log the result
	inGame, err := memory.IsInGame(d2r, globals.GetOffset("unitTable"))
	if err != nil {
		log.Fatalf("Failed to check if player is in-game: %v", err)
	}
	if inGame {
		log.Println("Player is currently in-game.")
	} else {
		log.Println("Player is not in-game.")
	}

	globals.InitSettings()

	// Create the overlay window
	processInfo := &utils.ProcessInfo{
		PID:     selectedProcess.PID,
		ExeName: selectedProcess.ExeName,
		Title:   selectedProcess.Title,
	}

	err = ui.CreateOverlayWindow(processInfo)
	if err != nil {
		log.Fatalf("Failed to create overlay window: %v", err)
	}

	// Initialize and start the overlay
	if err := ui.InitializeOverlay(); err != nil {
		log.Fatalf("Overlay initialization failed: %v", err)
	}
	go ui.RunOverlay()

	// Start memory reading routine
	go memory.ReadGameMemoryRoutine(d2r, cfg)

	// Run the message loop directly on the main thread
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 {
			// WM_QUIT received, exit the loop
			log.Println("Message loop received WM_QUIT, exiting.")
			break
		} else if ret == -1 {
			// Error occurred
			err := syscall.GetLastError()
			log.Printf("GetMessage error: %v", err)
			break
		} else {
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		}
	}

	// Signal overlay to close gracefully
	ui.CloseOverlay()
}
