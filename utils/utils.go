// utils/utils.go

package utils

import (
	"GalyMap/globals"
	"GalyMap/types"
	"fmt"
	"golang.org/x/sys/windows"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

// initializeAppLog sets up logging
func InitializeAppLog() {
	logFile, err := os.OpenFile("GalyMap.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error setting up log file:", err)
		return
	}
	log.SetOutput(logFile)
	log.Println("Application started")
}

// RemoveSpaces removes spaces from a hex string.
func RemoveSpaces(s string) string {
	result := ""
	for _, ch := range s {
		if ch != ' ' {
			result += string(ch)
		}
	}
	return result
}

// Helper function for safe type assertions
func ReadAndAssert[T any](d2r *ClassMemory, addr uintptr, dataType string, sizeBytes ...int) (T, error) {
	// Validate the data type
	validDataTypes := map[string]bool{
		"UChar":  true,
		"Char":   true,
		"UShort": true,
		"Short":  true,
		"UInt":   true,
		"Int":    true,
		"Float":  true,
		"Double": true,
		"Int64":  true,
		"String": true,
		"UInt64": true,
	}

	if !validDataTypes[dataType] {
		return *new(T), fmt.Errorf("invalid data type: %s", dataType)
	}

	var val interface{}
	var err error

	if dataType == "String" {
		// Default sizeBytes to 256 if not provided
		size := 256
		if len(sizeBytes) > 0 {
			size = sizeBytes[0]
		}
		val, err = d2r.ReadString(addr, size, "utf-8") // Adjust encoding as needed
	} else {
		val, err = d2r.Read(addr, dataType)
	}

	if err != nil {
		return *new(T), fmt.Errorf("failed to read at %x: %v", addr, err)
	}

	typedVal, ok := val.(T)
	if !ok {
		return *new(T), fmt.Errorf("failed to assert type at %x", addr)
	}
	return typedVal, nil
}

// ReadBufferAndAssert reads a value of the specified type from a byte slice at the given offset and performs a type assertion.
func ReadBufferAndAssert[T any](data []byte, offset int, dataType string) (T, error) {
	validDataTypes := map[string]bool{
		"UChar":  true,
		"Char":   true,
		"UShort": true,
		"Short":  true,
		"UInt":   true,
		"Int":    true,
		"Float":  true,
		"Double": true,
		"Int64":  true,
	}

	if !validDataTypes[dataType] {
		return *new(T), fmt.Errorf("invalid data type: %s", dataType)
	}
	val, err := ReadBuffer(data, offset, dataType)
	if err != nil {
		return *new(T), fmt.Errorf("failed to read buffer at offset %v: %v", offset, err)
	}
	typedVal, ok := val.(T)
	if !ok {
		return *new(T), fmt.Errorf("failed to assert type at offset %v", offset)
	}
	return typedVal, nil
}

// IfError checks if the error is not nil and prints the error message along with the file name and line number
func IfError(err error, message string) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("%s:%d: %s: %v\n", filepath.Base(file), line, message, err)
	}
}

// ReadNullTerminatedString converts a byte slice to a string, stopping at the first null byte.
func ReadNullTerminatedString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

// SetWindowPos sets the position and size of a window
func SetWindowPos(hwnd, targetHwnd windows.HWND, x, y, width, height int) error {
	modUser32 := windows.NewLazySystemDLL("user32.dll")
	procSetWindowPos := modUser32.NewProc("SetWindowPos")

	ret, _, err := procSetWindowPos.Call(
		uintptr(hwnd),
		uintptr(targetHwnd),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(0x0001|0x0002|0x0010), // SWP_NOACTIVATE | SWP_NOSIZE | SWP_NOZORDER
	)
	if ret == 0 {
		return fmt.Errorf("SetWindowPos failed: %v", err)
	}
	return nil
}

// FindWindow retrieves the HWND of a window based on its title
func FindWindow(title string) (windows.HWND, error) {
	modUser32 := windows.NewLazySystemDLL("user32.dll")
	procFindWindow := modUser32.NewProc("FindWindowW")

	titleUTF16, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, fmt.Errorf("failed to encode window title: %v", err)
	}

	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(titleUTF16)))
	if hwnd == 0 {
		return 0, fmt.Errorf("window not found: %s", title)
	}

	return windows.HWND(hwnd), nil
}

// GetPlayerPosition retrieves the player's position from GameMemoryData.
func GetPlayerPosition() (globals.UnitPosition, error) {
	globals.GameDataMutex.RLock()
	defer globals.GameDataMutex.RUnlock()

	x, okX := globals.GameMemoryData["xPos"].(float64)
	y, okY := globals.GameMemoryData["yPos"].(float64)
	if !okX || !okY {
		return globals.UnitPosition{}, fmt.Errorf("player position data missing or invalid")
	}

	return globals.UnitPosition{X: x, Y: y}, nil
}

// GetMobs retrieves the list of mobs from GameMemoryData.
func GetMobs() ([]globals.Mob, error) {
	globals.GameDataMutex.RLock()
	defer globals.GameDataMutex.RUnlock()

	mobs, ok := globals.GameMemoryData["mobs"].([]globals.Mob)
	if !ok {
		return nil, fmt.Errorf("mobs data missing or invalid")
	}

	return mobs, nil
}

// isUIOpen retrieves the menuShown value from GameMemoryData.
func IsUIOpen() (bool, error) {
	globals.GameDataMutex.RLock()
	defer globals.GameDataMutex.RUnlock()

	menuShown, ok := globals.GameMemoryData["menuShown"].(bool)
	if !ok {
		return false, fmt.Errorf("menuShown data missing or invalid")
	}

	return menuShown, nil
}

// getItems retrieves the list of items from GameMemoryData.
func GetItems() ([]types.Item, error) {
	globals.GameDataMutex.RLock()
	defer globals.GameDataMutex.RUnlock()

	items, ok := globals.GameMemoryData["items"].([]types.Item)
	if !ok {
		return nil, fmt.Errorf("items data missing or invalid")
	}

	return items, nil
}

// ItemFilter returns true or false base on whether the item meets item filter criteria.
func ItemFilter(item types.Item) bool {
	fmt.Println("ItemFilter: ", item.Name)
	return false
}
