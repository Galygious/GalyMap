package memory

import (
	"GalyMap/utils"
	// "log"
)

func IsInGame(d2r *utils.ClassMemory, startingOffset uintptr) (bool, error) {
	// Calculate the base address and read the unit table into a buffer
	baseAddress := uintptr(d2r.BaseAddress + startingOffset)
	// log.Printf("Base address: %v", baseAddress)
	unitTableBuffer, err := d2r.ReadRaw(baseAddress, 128*8) // Reading 128 Int64 values
	if err != nil {
		return false, err
	}

	// Iterate over each potential player unit in the unitTableBuffer
	for i := 0; i < 128; i++ {
		var playerUnitAddress, err = utils.ReadBufferAndAssert[int64](unitTableBuffer, 8*i, "Int64")
		utils.IfError(err, "Failed to read player unit address")
		// log.Printf("Player unit address: %v", playerUnitAddress)

		// Traverse the pointer chain if playerUnit is valid
		for playerUnitAddress > 0 {
			unitId, err := d2r.Read(uintptr(playerUnitAddress+0x08), "UInt")
			if err != nil {
				utils.IfError(err, "Error reading unitId")
				break
			}

			// Assert uint32 type for unitId
			if unitId, ok := unitId.(uint32); ok {
				pathAddress, err := d2r.Read(uintptr(playerUnitAddress+0x38), "Int64")
				if err != nil {
					utils.IfError(err, "Error reading pathAddress")
					break
				}

				// Assert int64 type for pathAddress
				if pathAddress, ok := pathAddress.(int64); ok {
					xPos, err := d2r.Read(uintptr(pathAddress)+0x02, "UShort")
					if err != nil {
						utils.IfError(err, "Error reading xPos")
						break
					}

					yPos, err := d2r.Read(uintptr(pathAddress)+0x06, "UShort")
					if err != nil {
						utils.IfError(err, "Error reading yPos")
						break
					}

					// Assert uint16 types for xPos and yPos
					if xPos, ok := xPos.(uint16); ok {
						if yPos, ok := yPos.(uint16); ok {
							// Check if unit is valid based on its position
							if unitId != 0 && xPos > 1 && yPos > 1 {
								return true, nil
							}

							// Move to the next player unit in the chain
							nextPlayerUnit, err := d2r.Read(uintptr(playerUnitAddress)+0x150, "Int64")
							if err != nil {
								utils.IfError(err, "Error following playerUnit chain")
								break
							}

							// Assert int64 type for nextPlayerUnit and update playerUnitAddress
							if nextPlayerUnit, ok := nextPlayerUnit.(int64); ok {
								playerUnitAddress = nextPlayerUnit
							} else {
								break
							}
						}
					}
				}
			}
		}
	}
	return false, nil
}
