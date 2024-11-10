// ui/process_selection.go

package ui

import (
	"GalyMap/memory"
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const (
	ID_LISTBOX = 101
	ID_BUTTON  = 102
)

var (
	selectedProcess *memory.Process
	processes       []memory.Process
	hInstance       win.HINSTANCE
)

func ShowProcessSelectionWindow(hInst win.HINSTANCE) (*memory.Process, error) {
	hInstance = hInst

	// Convert class name to UTF16 pointer
	className, err := syscall.UTF16PtrFromString("ProcessSelectionWindow")
	if err != nil {
		return nil, fmt.Errorf("failed to convert class name: %v", err)
	}

	// Register window class
	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpszClassName = className
	wc.LpfnWndProc = syscall.NewCallback(processSelectionWindowProc)
	wc.HInstance = hInstance
	wc.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW))
	wc.HbrBackground = win.HBRUSH(win.COLOR_WINDOW + 1)

	atom := win.RegisterClassEx(&wc)
	if atom == 0 {
		return nil, fmt.Errorf("failed to register window class: %v", syscall.GetLastError())
	}

	// Convert window title to UTF16 pointer
	windowTitle, err := syscall.UTF16PtrFromString("Select Diablo II Window")
	if err != nil {
		return nil, fmt.Errorf("failed to convert window title: %v", err)
	}

	// Create the window
	hwnd := win.CreateWindowEx(
		0,
		className,
		windowTitle,
		win.WS_OVERLAPPEDWINDOW,
		win.CW_USEDEFAULT,
		win.CW_USEDEFAULT,
		400,
		300,
		0,
		0,
		hInstance,
		nil,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("failed to create window: %v", syscall.GetLastError())
	}

	win.ShowWindow(hwnd, win.SW_SHOW)
	win.UpdateWindow(hwnd)

	// Run the message loop until the window is closed
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 {
			// WM_QUIT received, exit the loop
			break
		}
		if ret == -1 {
			log.Printf("GetMessage error: %v", syscall.GetLastError())
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}

	return selectedProcess, nil
}

func processSelectionWindowProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_CREATE:
		onCreate(hwnd)
		return 0
	case win.WM_COMMAND:
		onCommand(hwnd, wParam, lParam)
		return 0
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	default:
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}
}

func onCreate(hwnd win.HWND) {
	// Retrieve available processes
	var err error
	processes, err = memory.FindProcessesWithWindows()
	if err != nil || len(processes) == 0 {
		log.Println("No Diablo II window found.")
		msg, _ := syscall.UTF16PtrFromString("No Diablo II window found.")
		caption, _ := syscall.UTF16PtrFromString("Error")
		win.MessageBox(hwnd, msg, caption, win.MB_OK|win.MB_ICONERROR)
		win.DestroyWindow(hwnd)
		return
	}

	// Create a list box
	listBoxClass, err := syscall.UTF16PtrFromString("LISTBOX")
	if err != nil {
		log.Fatalf("Failed to convert class name: %v", err)
	}

	hListBox := win.CreateWindowEx(
		0,
		listBoxClass,
		nil,
		win.WS_CHILD|win.WS_VISIBLE|win.WS_BORDER|win.WS_VSCROLL|win.LBS_NOTIFY,
		10,
		10,
		360,
		200,
		hwnd,
		win.HMENU(ID_LISTBOX),
		hInstance,
		nil,
	)
	if hListBox == 0 {
		log.Printf("Failed to create list box: %v", syscall.GetLastError())
		return
	}

	// Populate the list box
	for i, proc := range processes {
		item := fmt.Sprintf("%s (%s)", proc.Title, proc.ExeName)
		itemPtr, err := syscall.UTF16PtrFromString(item)
		if err != nil {
			log.Printf("Failed to convert item string: %v", err)
			continue
		}
		index := win.SendMessage(hListBox, win.LB_ADDSTRING, 0, uintptr(unsafe.Pointer(itemPtr)))
		win.SendMessage(hListBox, win.LB_SETITEMDATA, index, uintptr(i))
	}

	// Create a button
	buttonClass, err := syscall.UTF16PtrFromString("BUTTON")
	if err != nil {
		log.Fatalf("Failed to convert class name: %v", err)
	}

	buttonText, err := syscall.UTF16PtrFromString("Select and Proceed")
	if err != nil {
		log.Fatalf("Failed to convert button text: %v", err)
	}

	hButton := win.CreateWindowEx(
		0,
		buttonClass,
		buttonText,
		win.WS_CHILD|win.WS_VISIBLE|win.BS_DEFPUSHBUTTON,
		150,
		220,
		100,
		30,
		hwnd,
		win.HMENU(ID_BUTTON),
		hInstance,
		nil,
	)
	if hButton == 0 {
		log.Printf("Failed to create button: %v", syscall.GetLastError())
		return
	}
}

func onCommand(hwnd win.HWND, wParam, lParam uintptr) {
	switch win.LOWORD(uint32(wParam)) {
	case ID_LISTBOX:
		if win.HIWORD(uint32(wParam)) == win.LBN_SELCHANGE {
			// Selection changed in the list box
			hListBox := win.GetDlgItem(hwnd, ID_LISTBOX)
			selIndex := int(win.SendMessage(hListBox, win.LB_GETCURSEL, 0, 0))
			if selIndex != -1 {
				selectedProcess = &processes[selIndex]
			}
		}
	case ID_BUTTON:
		if selectedProcess == nil {
			msg, _ := syscall.UTF16PtrFromString("No process selected.")
			caption, _ := syscall.UTF16PtrFromString("Warning")
			win.MessageBox(hwnd, msg, caption, win.MB_OK|win.MB_ICONWARNING)
			return
		}
		// Close the window and proceed
		win.DestroyWindow(hwnd)
	}
}
