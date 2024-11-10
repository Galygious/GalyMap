package memory

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32                    = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = modUser32.NewProc("EnumWindows")
	procGetWindowTextLength      = modUser32.NewProc("GetWindowTextLengthW")
	procGetWindowText            = modUser32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess              = modKernel32.NewProc("OpenProcess")
	modPsapi                     = syscall.NewLazyDLL("psapi.dll")
	procGetModuleFileNameEx      = modPsapi.NewProc("GetModuleFileNameExW")
)

type Process struct {
	Title   string
	ExeName string
	PID     uint32
}

func FindProcessesWithWindows() ([]Process, error) {
	var processes []Process
	callback := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		length, _, _ := procGetWindowTextLength.Call(uintptr(hwnd))
		if length > 0 {
			buffer := make([]uint16, length+1)
			procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buffer[0])), uintptr(length+1))
			title := syscall.UTF16ToString(buffer)

			if strings.Contains(strings.ToLower(title), "diablo ii: resurrected") {
				var pid uint32
				procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))

				exeName, err := getProcessImageFileName(pid)
				if err != nil {
					exeName = "Unknown"
				}

				processes = append(processes, Process{Title: title, ExeName: exeName, PID: pid})
			}
		}
		return 1
	})
	procEnumWindows.Call(callback, 0)
	return processes, nil
}

// getProcessImageFileName retrieves the executable name for a given PID using GetModuleFileNameEx.
func getProcessImageFileName(pid uint32) (string, error) {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(hProcess)

	var buf [windows.MAX_PATH]uint16
	ret, _, err := procGetModuleFileNameEx.Call(uintptr(hProcess), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return "", fmt.Errorf("failed to get module file name: %w", err)
	}

	fullPath := windows.UTF16ToString(buf[:])
	parts := strings.Split(fullPath, `\`)
	return parts[len(parts)-1], nil
}

func GetProcessHandle(pid uint32) (windows.Handle, error) {
	const PROCESS_ALL_ACCESS = 0x1F0FFF
	hProcess, _, err := procOpenProcess.Call(uintptr(PROCESS_ALL_ACCESS), 0, uintptr(pid))
	if hProcess == 0 {
		return 0, fmt.Errorf("could not open process: %v", err)
	}
	fmt.Printf("Opened process handle: %v\n", hProcess)
	return windows.Handle(hProcess), nil
}
