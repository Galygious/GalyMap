package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
)

func PatternScan(d2r *utils.ClassMemory) error {

	// unit table
	unitPattern := "48 03 C7 49 8B 8C C6"
	unitPatternAddress, err := d2r.ModulePatternScan("D2R.exe", unitPattern)
	if err != nil {
		return err
	}
	unitTable, err := d2r.Read(unitPatternAddress+7, "Int")
	if err != nil {
		return err
	}
	globals.Offsets.M["unitTable"] = uintptr(unitTable.(int32)) // Convert int32 to uintptr
	// log.Printf("Scanned and found unitTable offset: 0x%X", globals.Offsets.M["unitTable"])

	// ui
	uiPattern := "40 84 ed 0f 94 05"
	uiPatternAddress, err := d2r.ModulePatternScan("D2R.exe", uiPattern)
	if err != nil {
		return err
	}
	uiOffsetBuffer, err := d2r.Read(uiPatternAddress+6, "Int")
	if err != nil {
		return err
	}
	uiOffset := ((uiPatternAddress - d2r.BaseAddress) + 10 + uintptr(uiOffsetBuffer.(int32)))
	globals.Offsets.M["uiOffset"] = uiOffset
	// log.Printf("Scanned and found UI offset: 0x%X", uiOffset)

	// expansion
	expansionPattern := "48 8B 05 ?? ?? ?? ?? 48 8B D9 F3 0F 10 50 ??"
	expPatternAddress, err := d2r.ModulePatternScan("D2R.exe", expansionPattern)
	if err != nil {
		return err
	}
	expOffsetBuffer, err := d2r.Read(expPatternAddress+3, "Int")
	if err != nil {
		return err
	}
	expOffset := ((expPatternAddress - d2r.BaseAddress) + 7 + uintptr(expOffsetBuffer.(int32)))
	globals.Offsets.M["expOffset"] = expOffset
	// log.Printf("Scanned and found expansion offset: 0x%X", expOffset)

	// game data (IP and name)
	gameDataPattern := "44 88 25 ?? ?? ?? ?? 66 44 89 25 ?? ?? ?? ??"
	gameDataPatternAddress, err := d2r.ModulePatternScan("D2R.exe", gameDataPattern)
	if err != nil {
		return err
	}
	gameDataOffsetBuffer, err := d2r.Read(gameDataPatternAddress+0x3, "Int")
	if err != nil {
		return err
	}
	gameDataOffset := ((gameDataPatternAddress - d2r.BaseAddress) - 0x121 + uintptr(gameDataOffsetBuffer.(int32)))
	globals.Offsets.M["gameDataOffset"] = gameDataOffset
	// log.Printf("Scanned and found game data offset: 0x%X", gameDataOffset)

	// menu visibility
	menuPattern := "8B 05 ?? ?? ?? ?? 89 44 24 20 74 07"
	menuPatternAddress, err := d2r.ModulePatternScan("D2R.exe", menuPattern)
	if err != nil {
		return err
	}
	menuOffsetBuffer, err := d2r.Read(menuPatternAddress+2, "Int")
	if err != nil {
		return err
	}
	menuOffset := ((menuPatternAddress - d2r.BaseAddress) + 6 + uintptr(menuOffsetBuffer.(int32)))
	globals.Offsets.M["menuOffset"] = menuOffset
	// log.Printf("Scanned and found menu offset: 0x%X", menuOffset)

	// last hover object
	hoverPattern := "C6 84 C2 ?? ?? ?? ?? ?? 48 8B 74 24 ??"
	hoverPatternAddress, err := d2r.ModulePatternScan("D2R.exe", hoverPattern)
	if err != nil {
		return err
	}
	hoverOffset, err := d2r.Read(hoverPatternAddress+3, "Int")
	if err != nil {
		return err
	}
	globals.Offsets.M["hoverOffset"] = uintptr(hoverOffset.(int32)) - 1
	// log.Printf("Scanned and found hover offset: 0x%X", globals.Offsets.M["hoverOffset"])

	// roster
	rosterPattern := "02 45 33 D2 4D 8B"
	rosterPatternAddress, err := d2r.ModulePatternScan("D2R.exe", rosterPattern)
	if err != nil {
		return err
	}
	rosterOffsetBuffer, err := d2r.Read(rosterPatternAddress-3, "Int")
	if err != nil {
		return err
	}
	rosterOffset := ((rosterPatternAddress - d2r.BaseAddress) + 1 + uintptr(rosterOffsetBuffer.(int32)))
	globals.Offsets.M["rosterOffset"] = rosterOffset
	// log.Printf("Scanned and found roster offset: 0x%X", rosterOffset)

	return nil
}
