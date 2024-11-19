package memory

import (
	"GalyMap/globals"
	"GalyMap/types"
	"GalyMap/utils"
	// "log"
)

func ReadItems(d2r *utils.ClassMemory, startingOffset uintptr, itemAlertList map[string]bool) {
	// log.Printf("Reading items from offset 0x%x", startingOffset)
	baseAddress := d2r.BaseAddress + startingOffset + (4 * 1024)
	unitTableBuffer, err := d2r.ReadRaw(baseAddress, 128*8)
	utils.IfError(err, "Failed to read unit table buffer")

	// Clear items
	globals.Items = make([]types.Item, 0)

	// log.Printf("Beginning item read loop")
	for i := 0; i < 128; i++ {
		// log.Printf("Reading item %d", i)
		offset := i * 8
		itemUnit, err := utils.ReadBufferAndAssert[int64](unitTableBuffer, offset, "Int64")
		utils.IfError(err, "Failed to read item unit")
		for itemUnit > 0 {
			// log.Printf("Found valid item. Reading item unit 0x%x", itemUnit)
			itemStructData, err := d2r.ReadRaw(uintptr(itemUnit), 144)
			utils.IfError(err, "Failed to read item structure data")
			// log.Printf("Read item structure data")

			itemType, err := utils.ReadBufferAndAssert[uint32](itemStructData, 0x00, "UInt")
			utils.IfError(err, "Failed to read item type")
			// log.Printf("Read item type")

			txtFileNo, err := utils.ReadBufferAndAssert[uint32](itemStructData, 0x04, "UInt")
			utils.IfError(err, "Failed to read txtFileNo")
			// log.Printf("Read txtFileNo")

			itemLoc, err := utils.ReadBufferAndAssert[uint32](itemStructData, 0x0C, "UInt")
			utils.IfError(err, "Failed to read itemLoc")
			// log.Printf("Read itemLoc")

			if itemType == 4 && (itemLoc == 3 || itemLoc == 5) {
				// log.Printf("ItemType is 4 and itemLoc is 3 or 5")

				pUnitDataPtr, err := utils.ReadBufferAndAssert[int64](itemStructData, 0x10, "Int64")
				utils.IfError(err, "Failed to read pUnitDataPtr")
				// log.Printf("Read pUnitDataPtr")

				pUnitData, err := d2r.ReadRaw(uintptr(pUnitDataPtr), 144)
				utils.IfError(err, "Failed to read pUnitData")
				// log.Printf("Read pUnitData")

				flags, err := utils.ReadBufferAndAssert[uint32](pUnitData, 0x18, "UInt")
				utils.IfError(err, "Failed to read flags")
				// log.Printf("Read flags")

				itemQuality, err := utils.ReadBufferAndAssert[uint32](pUnitData, 0x00, "UInt")
				utils.IfError(err, "Failed to read itemQuality")
				// log.Printf("Read itemQuality")

				name := types.GetItemBaseName(int(txtFileNo))
				// log.Printf("Read name %s", name)

				if itemAlertList[name] || itemQuality > 2 {
					uniqueOrSetId, err := utils.ReadBufferAndAssert[uint32](pUnitData, 0x34, "UInt")
					utils.IfError(err, "Failed to read uniqueOrSetId")
					// log.Printf("Read uniqueOrSetId")

					pPathPtr, err := utils.ReadBufferAndAssert[int64](itemStructData, 0x38, "Int64")
					utils.IfError(err, "Failed to read pPathPtr")
					// log.Printf("Read pPathPtr")

					pPath, err := d2r.ReadRaw(uintptr(pPathPtr), 144)
					utils.IfError(err, "Failed to read pPath")
					// log.Printf("Read pPath")

					itemX, err := utils.ReadBufferAndAssert[uint16](pPath, 0x10, "UShort")
					utils.IfError(err, "Failed to read itemX")
					// log.Printf("Read itemX")

					itemY, err := utils.ReadBufferAndAssert[uint16](pPath, 0x14, "UShort")
					utils.IfError(err, "Failed to read itemY")
					// log.Printf("Read itemY")

					pStatsListExPtr, err := utils.ReadBufferAndAssert[int64](itemStructData, 0x88, "Int64")
					utils.IfError(err, "Failed to read pStatsListExPtr")
					// log.Printf("Read pStatsListExPtr")

					pStatsListEx, err := d2r.ReadRaw(uintptr(pStatsListExPtr), 180)
					utils.IfError(err, "Failed to read pStatsListEx")
					// log.Printf("Read pStatsListEx")

					statPtr, err := utils.ReadBufferAndAssert[int64](pStatsListEx, 0x30, "Int64")
					utils.IfError(err, "Failed to read statPtr")
					// log.Printf("Read statPtr")

					statCount, err := utils.ReadBufferAndAssert[uint32](pStatsListEx, 0x38, "UInt")
					utils.IfError(err, "Failed to read statCount")
					// log.Printf("Read statCount")

					statExPtr, err := utils.ReadBufferAndAssert[int64](pStatsListEx, 0x88, "Int64")
					utils.IfError(err, "Failed to read statExPtr")
					// log.Printf("Read statExPtr")

					statExCount, err := utils.ReadBufferAndAssert[uint32](pStatsListEx, 0x90, "UInt")
					utils.IfError(err, "Failed to read statExCount")
					// log.Printf("Read statExCount")

					item := types.NewItem(int(txtFileNo), int(itemQuality), int(uniqueOrSetId))
					item.Name = name
					item.ItemLoc = int(itemLoc)
					item.ItemX = int(itemX)
					item.ItemY = int(itemY)
					item.StatPtr = uintptr(statPtr)
					item.StatCount = int(statCount)
					item.StatExPtr = uintptr(statExPtr)
					item.StatExCount = int(statExCount)

					// log.Printf("Created item %s", item.Name)

					item.CalculateFlags(flags)
					// log.Printf("Calculated flags")

					globals.Items = append(globals.Items, *item)
					// log.Printf("Appended item to globals.Items")
				}
			}

			// log.Printf("Reading next item unit")

			nextItemUnit, err := utils.ReadAndAssert[int64](d2r, uintptr(itemUnit+0x150), "Int64")
			utils.IfError(err, "Failed to read next item unit")

			// log.Printf("Read next item unit")

			if nextItemUnit == itemUnit || nextItemUnit == 0 {
				// log.Printf("Breaking out of item loop")
				break
			}
			itemUnit = nextItemUnit
		}
	}
}
