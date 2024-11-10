// utils/memory.go

package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ClassMemory struct {
	BaseAddress          uintptr
	HProcess             windows.Handle
	PID                  uint32
	CurrentProgram       string
	InsertNullTerminator bool
	ReadStringLastError  bool
	IsTarget64bit        bool
	PtrType              string
	NumberOfBytesRead    uintptr
	NumberOfBytesWritten uintptr
}

var (
	aTypeSize = map[string]int{
		"UChar": 1, "Char": 1,
		"UShort": 2, "Short": 2,
		"UInt": 4, "Int": 4,
		"Float": 4, "Double": 8,
		"Int64": 8,
	}
	aRights = map[string]uint32{
		"PROCESS_ALL_ACCESS":                0x001F0FFF,
		"PROCESS_CREATE_PROCESS":            0x0080,
		"PROCESS_CREATE_THREAD":             0x0002,
		"PROCESS_DUP_HANDLE":                0x0040,
		"PROCESS_QUERY_INFORMATION":         0x0400,
		"PROCESS_QUERY_LIMITED_INFORMATION": 0x1000,
		"PROCESS_SET_INFORMATION":           0x0200,
		"PROCESS_SET_QUOTA":                 0x0100,
		"PROCESS_SUSPEND_RESUME":            0x0800,
		"PROCESS_TERMINATE":                 0x0001,
		"PROCESS_VM_OPERATION":              0x0008,
		"PROCESS_VM_READ":                   0x0010,
		"PROCESS_VM_WRITE":                  0x0020,
		"SYNCHRONIZE":                       0x00100000,
	}
)

const (
	PROCESSOR_ARCHITECTURE_INTEL = 0
	PROCESSOR_ARCHITECTURE_ARM64 = 12
	PROCESSOR_ARCHITECTURE_AMD64 = 9
	WAIT_TIMEOUT                 = 0x00000102
)

type SystemInfo struct {
	ProcessorArchitecture     uint16
	Reserved                  uint16
	PageSize                  uint32
	MinimumApplicationAddress uintptr
	MaximumApplicationAddress uintptr
	ActiveProcessorMask       uintptr
	NumberOfProcessors        uint32
	ProcessorType             uint32
	AllocationGranularity     uint32
	ProcessorLevel            uint16
	ProcessorRevision         uint16
}

func GetNativeSystemInfo(sysInfo *SystemInfo) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetNativeSystemInfo := kernel32.NewProc("GetNativeSystemInfo")
	procGetNativeSystemInfo.Call(uintptr(unsafe.Pointer(sysInfo)))
}

// ReadBuffer reads a value of the specified type from a byte slice at the given offset.
func ReadBuffer(data []byte, offset int, dataType string) (interface{}, error) {
	reader := bytes.NewReader(data[offset:])
	switch dataType {
	case "UChar":
		var val uint8
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Char":
		var val int8
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "UShort":
		var val uint16
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Short":
		var val int16
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "UInt":
		var val uint32
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Int":
		var val int32
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Float":
		var val float32
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Double":
		var val float64
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Int64":
		var val int64
		err := binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	default:
		return nil, fmt.Errorf("invalid data type")
	}
}

func NewClassMemory(program string, dwDesiredAccess uint32) (*ClassMemory, error) {
	cm := &ClassMemory{}
	cm.InsertNullTerminator = true

	fmt.Printf("Attempting to find PID for program: %s\n", program)
	pid, err := cm.findPID(program)
	if err != nil {
		fmt.Printf("Failed to find PID for program %s: %v\n", program, err)
		return nil, fmt.Errorf("process not found")
	}
	fmt.Printf("Found PID %d for program: %s\n", pid, program)
	cm.PID = pid

	if dwDesiredAccess == 0 {
		dwDesiredAccess = aRights["PROCESS_QUERY_INFORMATION"] | aRights["PROCESS_VM_OPERATION"] | aRights["PROCESS_VM_READ"] | aRights["PROCESS_VM_WRITE"] | aRights["SYNCHRONIZE"]
	}

	hProcess, err := windows.OpenProcess(dwDesiredAccess, false, pid)
	if err != nil {
		return nil, err
	}
	cm.HProcess = hProcess
	cm.IsTarget64bit, err = cm.isTargetProcess64Bit()
	if err != nil {
		return nil, err
	}
	if cm.IsTarget64bit {
		cm.PtrType = "Int64"
	} else {
		cm.PtrType = "UInt"
	}
	baseAddr, err := cm.getModuleBaseAddress("")
	if err != nil || baseAddr == 0 {
		baseAddr, err = cm.getProcessBaseAddress(program)
		if err != nil {
			return nil, err
		}
	}
	cm.BaseAddress = baseAddr
	return cm, nil
}

func (cm *ClassMemory) Close() error {
	return windows.CloseHandle(cm.HProcess)
}

// findPID locates the PID for the specified executable name.
func (cm *ClassMemory) findPID(program string) (uint32, error) {
	fmt.Printf("Searching for executable name: %s\n", program)
	pids := make([]uint32, 1024)
	var bytesReturned uint32
	err := windows.EnumProcesses(pids, &bytesReturned)
	if err != nil {
		return 0, err
	}
	numPids := bytesReturned / uint32(unsafe.Sizeof(pids[0]))
	for i := uint32(0); i < numPids; i++ {
		pid := pids[i]
		hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
		if err != nil {
			continue
		}
		defer windows.CloseHandle(hProcess)

		// Get the full path to the executable and extract only the filename
		fullPath, err := cm.getProcessImageFileName(hProcess)
		if err != nil {
			continue
		}
		exeName := strings.ToLower(fullPath[strings.LastIndex(fullPath, `\`)+1:])
		// fmt.Printf("Detected process: %s with PID %d. Checking against target: %s\n", exeName, pid, strings.ToLower(program))

		// Check if the extracted executable name matches the target program name
		if exeName == strings.ToLower(program) {
			fmt.Printf("Match found: PID %d for program %s\n", pid, program)
			return pid, nil
		}
	}
	return 0, errors.New("process not found")
}

func (cm *ClassMemory) getProcessImageFileName(hProcess windows.Handle) (string, error) {
	var hMod windows.Handle
	cbNeeded := uint32(0)
	err := windows.EnumProcessModules(hProcess, &hMod, uint32(unsafe.Sizeof(hMod)), &cbNeeded)
	if err != nil {
		return "", err
	}
	var szModName [windows.MAX_PATH]uint16
	err = windows.GetModuleFileNameEx(hProcess, hMod, &szModName[0], windows.MAX_PATH)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(szModName[:]), nil
}

func (cm *ClassMemory) isTargetProcess64Bit() (bool, error) {
	var sysInfo SystemInfo
	GetNativeSystemInfo(&sysInfo)
	if sysInfo.ProcessorArchitecture == PROCESSOR_ARCHITECTURE_INTEL {
		return false, nil
	}
	var wow64 bool
	err := windows.IsWow64Process(cm.HProcess, &wow64)
	if err != nil {
		return false, err
	}
	return !wow64, nil
}

func FixndWindow(className, windowName *uint16) (hwnd windows.HWND, err error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	procFindWindowW := user32.NewProc("FindWindowW")
	ret, _, err := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
	)
	if ret == 0 {
		if err != nil && err != syscall.Errno(0) {
			return 0, err
		}
		return 0, syscall.EINVAL
	}
	return windows.HWND(ret), nil
}

func (cm *ClassMemory) getModuleBaseAddress(moduleName string) (uintptr, error) {
	hMods := make([]windows.Handle, 1024)
	var cbNeeded uint32
	if err := windows.EnumProcessModules(cm.HProcess, &hMods[0], uint32(len(hMods))*uint32(unsafe.Sizeof(hMods[0])), &cbNeeded); err != nil {
		return 0, err
	}
	numMods := cbNeeded / uint32(unsafe.Sizeof(hMods[0]))
	for i := uint32(0); i < numMods; i++ {
		var modName [windows.MAX_PATH]uint16
		if err := windows.GetModuleBaseName(cm.HProcess, hMods[i], &modName[0], windows.MAX_PATH); err != nil {
			continue
		}
		name := windows.UTF16ToString(modName[:])
		if strings.EqualFold(name, moduleName) || moduleName == "" {
			return uintptr(hMods[i]), nil
		}
	}
	return 0, errors.New("module not found")
}

func (cm *ClassMemory) getProcessBaseAddress(program string) (uintptr, error) {
	hWnd, err := FindWindow(program)
	if err != nil {
		return 0, err
	}
	var pid uint32
	_, err = windows.GetWindowThreadProcessId(hWnd, &pid)
	if err != nil {
		return 0, err
	}
	if pid != cm.PID {
		return 0, errors.New("PID mismatch")
	}
	return cm.getModuleBaseAddress("")
}

func (cm *ClassMemory) Read(address uintptr, dataType string, offsets ...uintptr) (interface{}, error) {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return nil, err
	}
	var buf []byte
	size := aTypeSize[dataType]
	buf = make([]byte, size)
	var bytesRead uintptr
	err = windows.ReadProcessMemory(cm.HProcess, finalAddress, &buf[0], uintptr(size), &bytesRead)
	if err != nil {
		return nil, fmt.Errorf("failed to read process memory at address 0x%X: %w", finalAddress, err)
	}
	reader := bytes.NewReader(buf)
	switch dataType {
	case "UChar":
		var val uint8
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Char":
		var val int8
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "UShort":
		var val uint16
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Short":
		var val int16
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "UInt":
		var val uint32
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Int":
		var val int32
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Float":
		var val float32
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Double":
		var val float64
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	case "Int64":
		var val int64
		err = binary.Read(reader, binary.LittleEndian, &val)
		return val, err
	default:
		return nil, fmt.Errorf("invalid data type")
	}
}

func (cm *ClassMemory) ReadRaw(address uintptr, size uint32, offsets ...uintptr) ([]byte, error) {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	var bytesRead uintptr
	err = windows.ReadProcessMemory(cm.HProcess, finalAddress, &buf[0], uintptr(size), &bytesRead)
	if err != nil {
		return nil, err
	}
	return buf[:bytesRead], nil
}

func (cm *ClassMemory) ReadString(address uintptr, sizeBytes int, encoding string, offsets ...uintptr) (string, error) {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return "", err
	}
	if sizeBytes == 0 {
		sizeBytes = 256
	}
	buf := make([]byte, sizeBytes)
	var bytesRead uintptr
	err = windows.ReadProcessMemory(cm.HProcess, finalAddress, &buf[0], uintptr(sizeBytes), &bytesRead)
	if err != nil {
		cm.ReadStringLastError = true
		return "", err
	}
	cm.ReadStringLastError = false
	switch strings.ToLower(encoding) {
	case "utf-8":
		n := bytes.IndexByte(buf[:bytesRead], 0)
		if n >= 0 {
			return string(buf[:n]), nil
		}
		return string(buf[:bytesRead]), nil
	case "utf-16":
		u16 := make([]uint16, bytesRead/2)
		err = binary.Read(bytes.NewReader(buf[:bytesRead]), binary.LittleEndian, &u16)
		if err != nil {
			return "", err
		}
		n := 0
		for i, v := range u16 {
			if v == 0 {
				n = i
				break
			}
		}
		return string(utf16.Decode(u16[:n])), nil
	default:
		return "", fmt.Errorf("unsupported encoding")
	}
}

func (cm *ClassMemory) WriteString(address uintptr, data string, encoding string, offsets ...uintptr) error {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return err
	}
	var buf []byte
	switch strings.ToLower(encoding) {
	case "utf-8":
		buf = []byte(data)
	case "utf-16":
		u16 := utf16.Encode([]rune(data))
		var tmpBuf bytes.Buffer
		err = binary.Write(&tmpBuf, binary.LittleEndian, u16)
		if err != nil {
			return err
		}
		buf = tmpBuf.Bytes()
	default:
		return fmt.Errorf("unsupported encoding")
	}
	if cm.InsertNullTerminator {
		buf = append(buf, 0)
		if encoding == "utf-16" {
			buf = append(buf, 0)
		}
	}
	var bytesWritten uintptr
	err = windows.WriteProcessMemory(cm.HProcess, finalAddress, &buf[0], uintptr(len(buf)), &bytesWritten)
	return err
}

func (cm *ClassMemory) Write(address uintptr, value interface{}, dataType string, offsets ...uintptr) error {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, value)
	if err != nil {
		return err
	}
	var bytesWritten uintptr
	err = windows.WriteProcessMemory(cm.HProcess, finalAddress, &buf.Bytes()[0], uintptr(buf.Len()), &bytesWritten)
	return err
}

func (cm *ClassMemory) WriteRaw(address uintptr, data []byte, offsets ...uintptr) error {
	finalAddress, err := cm.calculateFinalAddress(address, offsets...)
	if err != nil {
		return err
	}
	var bytesWritten uintptr
	err = windows.WriteProcessMemory(cm.HProcess, finalAddress, &data[0], uintptr(len(data)), &bytesWritten)
	return err
}

func (cm *ClassMemory) WriteBytes(address uintptr, data interface{}, offsets ...uintptr) error {
	var buf []byte
	switch v := data.(type) {
	case string:
		pattern, err := cm.hexStringToPattern(v)
		if err != nil {
			return err
		}
		buf = pattern
	case []byte:
		buf = v
	default:
		return fmt.Errorf("unsupported data type")
	}
	return cm.WriteRaw(address, buf, offsets...)
}

func (cm *ClassMemory) calculateFinalAddress(address uintptr, offsets ...uintptr) (uintptr, error) {
	finalAddress := address
	var buf [8]byte
	for _, offset := range offsets {
		var bytesRead uintptr
		var err error
		if cm.IsTarget64bit {
			err = windows.ReadProcessMemory(cm.HProcess, finalAddress, &buf[0], 8, &bytesRead)
			if err != nil {
				return 0, err
			}
			finalAddress = uintptr(binary.LittleEndian.Uint64(buf[:8])) + offset
		} else {
			err = windows.ReadProcessMemory(cm.HProcess, finalAddress, &buf[0], 4, &bytesRead)
			if err != nil {
				return 0, err
			}
			finalAddress = uintptr(binary.LittleEndian.Uint32(buf[:4])) + offset
		}
	}
	return finalAddress, nil
}

func (cm *ClassMemory) hexStringToPattern(hexString string) ([]byte, error) {
	hexString = strings.ReplaceAll(hexString, " ", "")
	hexString = strings.ReplaceAll(hexString, "\t", "")
	hexString = strings.ReplaceAll(hexString, "0x", "")
	if len(hexString)%2 != 0 {
		return nil, errors.New("hex string has invalid length")
	}
	pattern := make([]byte, len(hexString)/2)
	for i := 0; i < len(hexString); i += 2 {
		byteStr := hexString[i : i+2]
		if byteStr == "??" {
			pattern[i/2] = '?'
		} else {
			var b byte
			_, err := fmt.Sscanf(byteStr, "%02X", &b)
			if err != nil {
				return nil, err
			}
			pattern[i/2] = b
		}
	}
	return pattern, nil
}

func (cm *ClassMemory) Suspend() error {
	ntdll := syscall.NewLazyDLL("ntdll.dll")
	ntSuspendProcess := ntdll.NewProc("NtSuspendProcess")
	r1, _, err := ntSuspendProcess.Call(uintptr(cm.HProcess))
	if r1 != 0 {
		return err
	}
	return nil
}

func (cm *ClassMemory) Resume() error {
	ntdll := syscall.NewLazyDLL("ntdll.dll")
	ntResumeProcess := ntdll.NewProc("NtResumeProcess")
	r1, _, err := ntResumeProcess.Call(uintptr(cm.HProcess))
	if r1 != 0 {
		return err
	}
	return nil
}

func (cm *ClassMemory) IsHandleValid() bool {
	result, _ := windows.WaitForSingleObject(cm.HProcess, 0)
	return result == WAIT_TIMEOUT
}

func (cm *ClassMemory) GetModuleInfo(moduleName string) (baseAddress uintptr, moduleSize uint32, err error) {
	hMods := make([]windows.Handle, 1024)
	var cbNeeded uint32
	if err := windows.EnumProcessModules(cm.HProcess, &hMods[0], uint32(len(hMods))*uint32(unsafe.Sizeof(hMods[0])), &cbNeeded); err != nil {
		return 0, 0, err
	}
	numMods := cbNeeded / uint32(unsafe.Sizeof(hMods[0]))
	for i := uint32(0); i < numMods; i++ {
		var modName [windows.MAX_PATH]uint16
		if err := windows.GetModuleBaseName(cm.HProcess, hMods[i], &modName[0], windows.MAX_PATH); err != nil {
			continue
		}
		name := windows.UTF16ToString(modName[:])
		if strings.EqualFold(name, moduleName) || (moduleName == "" && i == 0) {
			// Get module information
			var modInfo windows.ModuleInfo
			err := windows.GetModuleInformation(cm.HProcess, hMods[i], &modInfo, uint32(unsafe.Sizeof(modInfo)))
			if err != nil {
				return 0, 0, err
			}
			return uintptr(modInfo.BaseOfDll), modInfo.SizeOfImage, nil
		}
	}
	return 0, 0, errors.New("module not found")
}

func PatternScan(data []byte, pattern []byte) int {
	patternLength := len(pattern)
	dataLength := len(data)

	for i := 0; i <= dataLength-patternLength; i++ {
		match := true
		for j := 0; j < patternLength; j++ {
			if pattern[j] != '?' && data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func (cm *ClassMemory) ModulePatternScan(moduleName string, pattern string) (uintptr, error) {
	// Get the base address and size of the module
	baseAddress, moduleSize, err := cm.GetModuleInfo(moduleName)
	if err != nil {
		return 0, err
	}

	// Convert the pattern string into a byte pattern using your function
	needle, err := cm.hexStringToPattern(pattern)
	if err != nil {
		return 0, err
	}

	// Read the module's memory
	buffer, err := cm.ReadRaw(baseAddress, moduleSize)
	if err != nil {
		return 0, err
	}

	// Perform pattern scan on the buffer
	offset := PatternScan(buffer, needle)
	if offset == -1 {
		return 0, errors.New("pattern not found")
	}

	// Calculate the address where the pattern was found
	return baseAddress + uintptr(offset), nil
}

// func main() {
// 	cm, err := NewClassMemory("notepad.exe", 0)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	defer cm.Close()

// 	fmt.Printf("Base Address: 0x%X\n", cm.BaseAddress)

// 	// Example usage
// 	value, err := cm.Read(0x12345678, "UInt")
// 	if err != nil {
// 		fmt.Println("Read error:", err)
// 	} else {
// 		fmt.Printf("Value: %v\n", value)
// 	}

// 	err = cm.Write(0x12345678, uint32(1234), "UInt")
// 	if err != nil {
// 		fmt.Println("Write error:", err)
// 	} else {
// 		fmt.Println("Value written successfully")
// 	}
// }
