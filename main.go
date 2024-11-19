// main.go
package main

import (
	"GalyMap/config"
	"GalyMap/globals"
	"GalyMap/memory"
	"GalyMap/types"
	"GalyMap/ui"
	"GalyMap/utils"
	"log"
	"runtime"
	"syscall"

	"github.com/lxn/win"
)

func main() {
	// Lock the main goroutine to its OS thread
	// This is crucial for GLFW to function correctly
	// Ensure that no other goroutines perform GLFW operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Initialize global settings and maps
	globals.InitSettings()

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

	nipsFolderPath := "./config/nips/"
	err = types.LoadNipRules(nipsFolderPath)
	if err != nil {
		log.Fatalf("Failed to load NIP rules: %v", err)
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

	// Perform Pattern Scan
	err = memory.PatternScan(d2r)
	if err != nil {
		log.Fatalf("Pattern scan failed: %v", err)
	}

	// Check if player is in-game and log the result
	unitTableOffset, exists := globals.GetOffset("unitTable")
	if !exists {
		log.Fatalf("main.go: Failed to retrieve 'unitTable' offset from globals.")
	}
	inGame, err := memory.IsInGame(d2r, unitTableOffset)
	if err != nil {
		log.Fatalf("Failed to check if player is in-game: %v", err)
	}
	if inGame {
		log.Println("Player is currently in-game.")
	} else {
		log.Println("Player is not in-game.")
	}

	// Create the overlay window information
	processInfo := &globals.ProcessInfo{
		PID:     selectedProcess.PID,
		ExeName: selectedProcess.ExeName,
		Title:   selectedProcess.Title,
	}

	// Initialize and create the overlay
	if err := ui.InitializeOverlay(processInfo, cfg); err != nil {
		log.Fatalf("Overlay initialization failed: %v", err)
	}

	// Start memory reading routine in a separate goroutine
	go ui.ReadGameMemoryRoutine(d2r, cfg)

	// Run the overlay render loop on the main goroutine
	err = ui.RunOverlay()
	if err != nil {
		log.Fatalf("Overlay run failed: %v", err)
	}

	// After RunOverlay exits, continue with shutdown
	log.Println("Main program execution completed.")
}
