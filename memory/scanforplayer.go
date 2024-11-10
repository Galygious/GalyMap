package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
	"fmt"
	"log"
	"strings"
)

var lastPlayerPointer uintptr

func ScanForPlayer(d2r *utils.ClassMemory, startingOffset uintptr) uintptr {
	if CheckPlayerPointer(d2r, lastPlayerPointer) {
		return lastPlayerPointer
	} else {
		log.Printf("Scanning for new player pointer %v, starting default offset %v", lastPlayerPointer, startingOffset)
		log.Printf("base address: 0x%x", d2r.BaseAddress)
		log.Printf("unit table offset: 0x%x", startingOffset)
		log.Printf("unit table address: 0x%x", d2r.BaseAddress+startingOffset)
		lastPlayerPointer = GetPlayerOffset(d2r, startingOffset)
	}
	return lastPlayerPointer
}

func CheckPlayerPointer(d2r *utils.ClassMemory, playerUnit uintptr) bool {
	if playerUnit == 0 {
		log.Println("Player unit pointer is null.")
		return false
	}

	// Read actAddress
	pAct := playerUnit + 0x20
	actAddress, err := utils.ReadAndAssert[uintptr](d2r, pAct, "Int64")
	if err != nil || actAddress == 0 {
		log.Printf("Error reading act address at 0x%X: %v", pAct, err)
		return false
	}

	// Read mapSeed
	mapSeedAddress := actAddress + 0x1C
	mapSeed, err := utils.ReadAndAssert[uint32](d2r, mapSeedAddress, "UInt")
	if err != nil || mapSeed == 0 {
		log.Printf("Error reading map seed at 0x%X: %v", mapSeedAddress, err)
		return false
	}

	// Read pathAddress
	pPath := playerUnit + 0x38
	pathAddress, err := utils.ReadAndAssert[uintptr](d2r, pPath, "Int64")
	if err != nil || pathAddress == 0 {
		log.Printf("Error reading path address at 0x%X: %v", pPath, err)
		return false
	}

	// Read xPos
	xPosAddress := pathAddress + 0x02
	xPos, err := utils.ReadAndAssert[uint16](d2r, xPosAddress, "UShort")
	if err != nil || xPos == 0 {
		log.Printf("Error reading x position at 0x%X: %v", xPosAddress, err)
		return false
	}

	// Read yPos
	yPosAddress := pathAddress + 0x06
	yPos, err := utils.ReadAndAssert[uint16](d2r, yPosAddress, "UShort")
	if err != nil || yPos == 0 {
		log.Printf("Error reading y position at 0x%X: %v", yPosAddress, err)
		return false
	}

	// If all values are valid
	log.Printf("Player pointer valid. xPos: %d, yPos: %d, mapSeed: %d", xPos, yPos, mapSeed)
	return true
}

func GetPlayerOffset(d2r *utils.ClassMemory, startingOffset uintptr) uintptr {
	for attempts := 1; attempts <= 128; attempts++ {
		newOffset := startingOffset + uintptr((attempts-1)*8)
		startingAddress := d2r.BaseAddress + newOffset

		playerUnit, err := utils.ReadAndAssert[int64](d2r, startingAddress, "Int64")
		utils.IfError(err, "Error reading player unit address")
		for playerUnit > 0 {
			pInventory := playerUnit + 0x90
			inventory, err := utils.ReadAndAssert[int64](d2r, uintptr(pInventory), "Int64")
			utils.IfError(err, "Error reading inventory address")
			if inventory != 0 {
				expCharPtr, err := utils.ReadAndAssert[int64](d2r, d2r.BaseAddress+globals.Offsets.M["expOffset"], "Int64")
				utils.IfError(err, "Error reading expCharPtr")
				expChar, err := utils.ReadAndAssert[uint16](d2r, uintptr(expCharPtr+0x5C), "UShort")
				utils.IfError(err, "Error reading expChar")
				var basecheck bool
				val, err := utils.ReadAndAssert[uint16](d2r, uintptr(inventory+0x30), "UShort")
				utils.IfError(err, "Error reading basecheck 1")
				basecheck = val != 1
				if expChar != 0 {
					val, err := utils.ReadAndAssert[uint16](d2r, uintptr(inventory+0x70), "UShort")
					utils.IfError(err, "Error reading basecheck 1")
					basecheck = val != 0
				}

				if basecheck {
					pAct := playerUnit + 0x20
					actAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(pAct), "Int64")
					utils.IfError(err, "Error reading act address")
					mapSeedAddress := actAddress + 0x1C
					mapSeed, err := utils.ReadAndAssert[uint32](d2r, uintptr(mapSeedAddress), "UInt")
					utils.IfError(err, "Error reading map seed")

					pPath := playerUnit + 0x38
					pathAddress, _ := utils.ReadAndAssert[int64](d2r, uintptr(pPath), "Int64")
					xPos, _ := utils.ReadAndAssert[uint16](d2r, uintptr(pathAddress+0x02), "UShort")
					yPos, _ := utils.ReadAndAssert[uint16](d2r, uintptr(pathAddress+0x06), "UShort")

					pUnitData := playerUnit + 0x10
					playerNameAddress, _ := utils.ReadAndAssert[int64](d2r, uintptr(pUnitData), "Int64")
					var name strings.Builder
					for i := 0; i < 16; i++ {
						char, _ := utils.ReadAndAssert[byte](d2r, uintptr(playerNameAddress+int64(i)), "UChar")
						name.WriteByte(char)
					}

					if xPos > 0 && yPos > 0 && len(fmt.Sprintf("%v", mapSeed)) > 6 {
						// log.Printf("SUCCESS: Found current player offset: %v, %v %v at entry %v, which gives obfuscated map seed: %v", newOffset, xPos, yPos, attempts, mapSeed)
						return uintptr(playerUnit)
					}
				}
			}
			playerUnit, err = utils.ReadAndAssert[int64](d2r, uintptr(playerUnit+0x150), "Int64")
			utils.IfError(err, "Error reading next player unit")
		}
	}
	log.Println("Did not find a player offset in unit hashtable, likely in game menu.")
	return 0
}
