package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
	// "log"
)

func ReadParty(d2r *utils.ClassMemory, playerUnitId uint32) {
	rosterOffset := globals.Offsets.M["rosterOffset"]
	baseAddress := d2r.BaseAddress + rosterOffset
	partyStruct, err := utils.ReadAndAssert[int64](d2r, uintptr(baseAddress), "Int64")
	utils.IfError(err, "Error reading partyStruct")

	for partyStruct > 0 {
		// log.Printf("in party loop")
		name, err := utils.ReadAndAssert[string](d2r, uintptr(partyStruct), "String", 16)
		utils.IfError(err, "Error reading name")
		unitId, err := utils.ReadAndAssert[uint32](d2r, uintptr(partyStruct+0x48), "UInt")
		utils.IfError(err, "Error reading unitId")
		area, err := utils.ReadAndAssert[uint32](d2r, uintptr(partyStruct+0x5C), "UInt")
		utils.IfError(err, "Error reading area")
		plevel, err := utils.ReadAndAssert[uint16](d2r, uintptr(partyStruct+0x58), "UShort")
		utils.IfError(err, "Error reading plevel")
		partyId, err := utils.ReadAndAssert[uint16](d2r, uintptr(partyStruct+0x5A), "UShort")
		utils.IfError(err, "Error reading partyId")
		xPos, err := utils.ReadAndAssert[uint32](d2r, uintptr(partyStruct+0x60), "UInt")
		utils.IfError(err, "Error reading xPos")
		yPos, err := utils.ReadAndAssert[uint32](d2r, uintptr(partyStruct+0x64), "UInt")
		utils.IfError(err, "Error reading yPos")
		// hostilePtr, err := utils.ReadAndAssert[int64](d2r, uintptr(partyStruct+0x70), "Int64")
		// utils.IfError(err, "Error reading hostilePtr")

		isHostileToPlayer := false

		// for hostilePtr > 0 {
		// 	log.Printf("in hostile loop")
		// 	hostileUnitId, err := utils.ReadAndAssert[uint32](d2r, uintptr(hostilePtr), "UInt")
		// 	utils.IfError(err, "Error reading hostileUnitId")
		// 	hostileFlag, err := utils.ReadAndAssert[uint32](d2r, uintptr(hostilePtr+0x04), "UInt")
		// 	utils.IfError(err, "Error reading hostileFlag")
		// 	hostilePtr, err = utils.ReadAndAssert[int64](d2r, uintptr(hostilePtr+0x08), "Int64")
		// 	utils.IfError(err, "Error reading hostilePtr")
		// 	if playerUnitId == hostileUnitId {
		// 		if hostileFlag > 0 {
		// 			isHostileToPlayer = true
		// 		}
		// 	}
		// }

		player := globals.Player{
			Name:              name,
			UnitId:            unitId,
			Area:              area,
			PartyId:           partyId,
			Plevel:            plevel,
			Pos:               globals.UnitPosition{X: float64(xPos), Y: float64(yPos)},
			IsHostileToPlayer: isHostileToPlayer,
		}

		globals.PartyList = append(globals.PartyList, player)
		partyStruct, err = utils.ReadAndAssert[int64](d2r, uintptr(partyStruct+0x148), "Int64")
		utils.IfError(err, "Error reading partyStruct")
	}
	// log.Printf("Party list: %v", globals.PartyList)
}
