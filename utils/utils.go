package utils

import (
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
