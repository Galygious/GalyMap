package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
)

// ReadOtherPlayers reads the other players' data from memory and updates the global OtherPlayers slice.
// Parameters:
// - d2r: a pointer to the ClassMemory instance used to read memory.
// - startingOffset: the offset from the base address to start reading.
// - levelNo: the current level number (not used in this function).
// - partyList: a list of players in the party.
func ReadOtherPlayers(d2r *utils.ClassMemory, startingOffset uintptr, levelNo int, partyList []utils.Player) {
	globals.OtherPlayers = []utils.Player{}

	baseAddress := d2r.BaseAddress + startingOffset
	unitTableBuffer, err := d2r.ReadRaw(baseAddress, 128*8)
	utils.IfError(err, "Error reading unit table buffer")

	for i := 0; i < 128; i++ {
		offset := i * 8
		playerUnitAddress, err := utils.ReadBufferAndAssert[int64](unitTableBuffer, offset, "Int64")
		utils.IfError(err, "Failed to read player unit address")
		if playerUnitAddress == 0 {
			continue
		}

		processPlayerUnits(d2r, playerUnitAddress, i)
	}

	existingPlayers := make(map[uint32]bool)
	for _, unitPlayer := range globals.OtherPlayers {
		existingPlayers[unitPlayer.UnitId] = true
	}

	for _, partyPlayer := range partyList {
		if !existingPlayers[partyPlayer.UnitId] && partyPlayer.Area == uint32(levelNo) {
			globals.OtherPlayers = append(globals.OtherPlayers, utils.Player{
				Name:       partyPlayer.Name,
				UnitId:     partyPlayer.UnitId,
				Pos:        partyPlayer.Pos,
				IsCorpse:   false,
				Player:     len(globals.OtherPlayers) + 1,
				PlayerName: partyPlayer.Name,
			})
		}
	}
}

func processPlayerUnits(d2r *utils.ClassMemory, playerUnitAddress int64, playerIndex int) {
	for playerUnitAddress != 0 {
		inventoryAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(playerUnitAddress)+0x90, "Int64")
		utils.IfError(err, "Failed to read inventory address")

		if inventoryAddress != 0 {
			unitID, err := utils.ReadAndAssert[uint32](d2r, uintptr(playerUnitAddress)+0x08, "UInt")
			utils.IfError(err, "Failed to read unitID")

			pathAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(playerUnitAddress)+0x38, "Int64")
			utils.IfError(err, "Failed to read unitID")

			xPosFloat, yPosFloat, err := readPlayerPosition(d2r, pathAddress)
			if err != nil {
				utils.IfError(err, "Failed to read player position")
				continue
			}

			playerNameAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(playerUnitAddress)+0x10, "Int64")
			utils.IfError(err, "Failed to read player name address")

			playerName, err := d2r.ReadString(uintptr(playerNameAddress), 0, "utf-8")
			utils.IfError(err, "Error reading player name")

			isCorpseValue, err := utils.ReadAndAssert[uint8](d2r, uintptr(playerUnitAddress)+0x1A6, "UChar")
			utils.IfError(err, "Failed to read isCorpse value")
			isCorpse := isCorpseValue == 1

			if xPosFloat > 1 && yPosFloat > 1 {
				globals.OtherPlayers = append(globals.OtherPlayers, utils.Player{
					Name:       playerName,
					UnitId:     unitID,
					Pos:        utils.UnitPosition{X: xPosFloat, Y: yPosFloat},
					IsCorpse:   isCorpse,
					Player:     playerIndex + 1,
					PlayerName: playerName,
				})
			}
		}

		nextPlayerUnitAddress, err := utils.ReadAndAssert[int64](d2r, uintptr(playerUnitAddress)+0x150, "Int64")
		utils.IfError(err, "Failed to read next player unit address")
		if nextPlayerUnitAddress == playerUnitAddress || nextPlayerUnitAddress == 0 {
			break
		}
		playerUnitAddress = nextPlayerUnitAddress
	}
}

func readPlayerPosition(d2r *utils.ClassMemory, pathAddress int64) (float64, float64, error) {
	pathBufferSize := uintptr(0x08)
	pathBuffer, err := d2r.ReadRaw(uintptr(pathAddress), uint32(pathBufferSize))
	if err != nil {
		return 0, 0, err
	}

	xPosOffset, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x00, "UShort")
	if err != nil {
		return 0, 0, err
	}
	xPos, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x02, "UShort")
	if err != nil {
		return 0, 0, err
	}
	yPosOffset, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x04, "UShort")
	if err != nil {
		return 0, 0, err
	}
	yPos, err := utils.ReadBufferAndAssert[uint16](pathBuffer, 0x06, "UShort")
	if err != nil {
		return 0, 0, err
	}

	xPosFloat := float64(xPos) + float64(xPosOffset)/65536.0
	yPosFloat := float64(yPos) + float64(yPosOffset)/65536.0

	return xPosFloat, yPosFloat, nil
}
