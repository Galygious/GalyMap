package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
	"log"
	"syscall"
)

var (
	lastdwInitSeedHash1 uint32
	lastdwInitSeedHash2 uint32
	lastdwEndSeedHash1  uint32
	xorkey              uint32
	playerLevel         uint32
	experience          uint32
	modRustDecrypt      = syscall.NewLazyDLL("rustdecrypt.dll")
	procGetSeed         = modRustDecrypt.NewProc("get_seed")
	partyList           []utils.Player
	lastHoveredType     uint32
	lastHoveredUnitId   uint32
)

func ReadGameMemory(d2r *utils.ClassMemory, settings map[string]bool) {
	playerPointer := ScanForPlayer(d2r, globals.Offsets.M["unitTable"])

	playerUnit := playerPointer
	unitId, err := utils.ReadAndAssert[uint32](d2r, playerUnit+0x08, "UInt")
	utils.IfError(err, "Failed to read unitId")

	pPath, err := utils.ReadAndAssert[int64](d2r, playerUnit+0x38, "Int64")
	utils.IfError(err, "Failed to read pPath")
	pRoom1Address, err := utils.ReadAndAssert[int64](d2r, uintptr(pPath+0x20), "Int64")
	utils.IfError(err, "Failed to read pRoom1Address")
	pRoom2Address, err := utils.ReadAndAssert[int64](d2r, uintptr(pRoom1Address+0x18), "Int64")
	utils.IfError(err, "Failed to read pRoom2Address")
	pLevelAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(pRoom2Address+0x90), "Int64")
	utils.IfError(err, "Failed to read pLevelAddress")
	levelNo, err := utils.ReadAndAssert[uint32](d2r, uintptr(pLevelAddress+0x1F8), "UInt")
	utils.IfError(err, "Failed to read levelNo")

	playerNameAddress, err := utils.ReadAndAssert[int64](d2r, playerUnit+0x10, "Int64")
	utils.IfError(err, "Failed to read playerNameAddress")
	playerName, err := d2r.ReadString(uintptr(playerNameAddress), 0, "utf-8")
	utils.IfError(err, "Failed to read playerName")

	actAddress, err := utils.ReadAndAssert[int64](d2r, playerUnit+0x20, "Int64")
	utils.IfError(err, "Failed to read actAddress")
	actMiscAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(actAddress+0x78), "Int64")
	utils.IfError(err, "Failed to read actMiscAddress")

	dwInitSeedHash1, err := utils.ReadAndAssert[uint32](d2r, uintptr(actMiscAddress+0x840), "UInt")
	utils.IfError(err, "Failed to read dwInitSeedHash1")
	dwInitSeedHash2, err := utils.ReadAndAssert[uint32](d2r, uintptr(actMiscAddress+0x844), "UInt")
	utils.IfError(err, "Failed to read dwInitSeedHash2")
	dwEndSeedHash1, err := utils.ReadAndAssert[uint32](d2r, uintptr(actMiscAddress+0x868), "UInt")
	utils.IfError(err, "Failed to read dwEndSeedHash1")

	if dwInitSeedHash1 != lastdwInitSeedHash1 || dwInitSeedHash2 != lastdwInitSeedHash2 || globals.MapSeed == 0 {
		globals.MapSeed = calculateMapSeed(dwInitSeedHash1, dwInitSeedHash2, dwEndSeedHash1)
		lastdwInitSeedHash1 = dwInitSeedHash1
		lastdwInitSeedHash2 = dwInitSeedHash2
	}

	aActUnk2, err := utils.ReadAndAssert[int64](d2r, uintptr(actAddress+0x78), "Int64")
	utils.IfError(err, "Failed to read aActUnk2")
	difficulty, err := utils.ReadAndAssert[uint16](d2r, uintptr(aActUnk2+0x830), "UShort")
	utils.IfError(err, "Failed to read difficulty")

	if globals.Ticktock%6 == 0 {
		pStatsListEx, err := utils.ReadAndAssert[int64](d2r, playerUnit+0x88, "Int64")
		utils.IfError(err, "Failed to read pStatsListEx")
		statPtr, err := utils.ReadAndAssert[int64](d2r, uintptr(pStatsListEx+0x30), "Int64")
		utils.IfError(err, "Failed to read statPtr")
		statCount, err := utils.ReadAndAssert[int64](d2r, uintptr(pStatsListEx+0x38), "Int64")
		utils.IfError(err, "Failed to read statCount")
		buffer, err := d2r.ReadRaw(uintptr(statPtr+0x2), uint32(statCount*8))
		utils.IfError(err, "Failed to read raw stats")

		for i := 0; i < int(statCount); i++ {
			offset := i * 8
			statEnum, err := utils.ReadBufferAndAssert[uint16](buffer, offset, "UShort")
			utils.IfError(err, "Failed to get statEnum")
			statValue, err := utils.ReadBufferAndAssert[uint32](buffer, offset+0x2, "UInt")
			utils.IfError(err, "Failed to get statValue")
			if statEnum == 12 {
				playerLevel = statValue
			}
			if statEnum == 13 {
				experience = statValue
			}
		}
	}

	hoverAddress := d2r.BaseAddress + globals.Offsets.M["hoverOffset"]
	hoverBuffer, err := d2r.ReadRaw(uintptr(hoverAddress), 12)
	utils.IfError(err, "Failed to read hoverBuffer")
	isHovered, err := utils.ReadBufferAndAssert[uint8](hoverBuffer, 0, "UChar")
	utils.IfError(err, "Failed to get isHovered")
	if isHovered != 0 {
		lastHoveredType, err = utils.ReadBufferAndAssert[uint32](hoverBuffer, 0x04, "UInt")
		utils.IfError(err, "Failed to get lastHoveredType")
		lastHoveredUnitId, err = utils.ReadBufferAndAssert[uint32](hoverBuffer, 0x08, "UInt")
		utils.IfError(err, "Failed to get lastHoveredUnitId")
	}

	if globals.Ticktock%3 == 0 {
		ReadParty(d2r, unitId)
	}

	if settings["showOtherPlayers"] {
		ReadOtherPlayers(d2r, globals.Offsets.M["unitTable"], int(levelNo), partyList)
	}

	if settings["showNormalMobs"] || settings["showUniqueMobs"] || settings["showBosses"] || settings["showDeadMobs"] {
		if lastHoveredType != 0 {
			ReadMobs(d2r, globals.Offsets.M["unitTable"], lastHoveredUnitId)
		} else {
			ReadMobs(d2r, globals.Offsets.M["unitTable"], 0)
		}
	}

	missiles := []interface{}{}
	if settings["showPlayerMissiles"] {
		playerMissiles, err := ReadMissiles(d2r, int(globals.Offsets.M["unitTable"]+(6*1024)))
		utils.IfError(err, "Failed to read playerMissiles")
		missiles = append(missiles, playerMissiles)
	}

	if settings["showEnemyMissiles"] {
		enemyMissiles, err := ReadMissiles(d2r, int(globals.Offsets.M["unitTable"]))
		utils.IfError(err, "Failed to read enemyMissiles")
		missiles = append(missiles, enemyMissiles)
	}

	if settings["enableItemFilter"] && globals.Ticktock%3 == 0 {
		ReadItems(d2r, globals.Offsets.M["unitTable"], globals.ItemAlertList)
	}

	if settings["showShrines"] || settings["showPortals"] || settings["showChests"] && globals.Ticktock%6 == 0 {
		if lastHoveredType == 2 {
			ReadObjects(d2r, int(globals.Offsets.M["unitTable"]), lastHoveredUnitId, int(levelNo))
		} else {
			ReadObjects(d2r, int(globals.Offsets.M["unitTable"]), 0, int(levelNo))
		}
	}

	menuShown, err := ReadUI(d2r)
	utils.IfError(err, "Failed to read UI")

	pathAddress, err := utils.ReadAndAssert[int64](d2r, playerUnit+0x38, "Int64")
	utils.IfError(err, "Failed to read pathAddress")

	pathBuffer, err := d2r.ReadRaw(uintptr(pathAddress), 8)
	utils.IfError(err, "Failed to read path buffer")

	xPosOffset, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x00, "UShort")
	utils.IfError(err, "Failed to read xPosOffset")
	xPosint, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x02, "UShort")
	utils.IfError(err, "Failed to read xPos")
	yPosOffset, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x04, "UShort")
	utils.IfError(err, "Failed to read yPosOffset")
	yPosint, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x06, "UShort")
	utils.IfError(err, "Failed to read yPos")
	xPos := float64(xPosint) + float64(xPosOffset)/65536.0
	yPos := float64(yPosint) + float64(yPosOffset)/65536.0

	if xPos == 0 {
		log.Printf("Did not find player position at player offset %v", globals.Offsets.M["unitTable"])
	}

	globals.GameMemoryData["playerPointer"] = playerPointer
	globals.GameMemoryData["pathAddress"] = pathAddress
	// globals.GameMemoryData["gameName"] = gameName
	globals.GameMemoryData["mapSeed"] = globals.MapSeed
	globals.GameMemoryData["difficulty"] = difficulty
	globals.GameMemoryData["levelNo"] = levelNo
	globals.GameMemoryData["xPos"] = xPos
	globals.GameMemoryData["yPos"] = yPos
	globals.GameMemoryData["mobs"] = globals.Mobs
	globals.GameMemoryData["missiles"] = missiles
	globals.GameMemoryData["otherPlayers"] = globals.OtherPlayers
	globals.GameMemoryData["items"] = globals.Items
	globals.GameMemoryData["objects"] = globals.GameObjects
	globals.GameMemoryData["playerName"] = playerName
	globals.GameMemoryData["experience"] = experience
	globals.GameMemoryData["playerLevel"] = playerLevel
	globals.GameMemoryData["menuShown"] = menuShown
	globals.GameMemoryData["hoveredMob"] = globals.HoveredMob
	globals.GameMemoryData["partyList"] = partyList
	globals.GameMemoryData["unitId"] = unitId
}

func calculateMapSeed(InitSeedHash1, InitSeedHash2, EndSeedHash1 uint32) uint32 {
	// log.Printf("Calculating new map seed from %v %v %v", InitSeedHash1, InitSeedHash2, EndSeedHash1)

	// Call the DLL function
	ret, _, err := procGetSeed.Call(
		uintptr(InitSeedHash1),
		uintptr(InitSeedHash2),
		uintptr(EndSeedHash1),
	)
	if err != nil && err.Error() != "The operation completed successfully." {
		log.Printf("Failed to call get_seed: %v", err)
		return 0
	}

	mapSeed := uint32(ret)
	// log.Printf("Found mapSeed '%v'", mapSeed)
	if mapSeed == 0 {
		log.Printf("ERROR: YOU HAVE AN ERROR DECRYPTING THE MAP SEED, YOUR MAPS WILL EITHER NOT APPEAR OR NOT LINE UP")
	}
	return mapSeed
}
