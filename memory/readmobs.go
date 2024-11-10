package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
	"encoding/binary"
	// "log"
)

// ReadMobs reads mobs from the game process and updates the global mobs list and hoveredMob
func ReadMobs(d2r *utils.ClassMemory, startingOffset uintptr, currentHoveringUnitId uint32) {

	// log.Printf("Reading mobs")
	globals.Mobs = []utils.Mob{}     // Initialize the global mobs list
	globals.HoveredMob = utils.Mob{} // Initialize the global hoveredMob

	baseAddress := d2r.BaseAddress + startingOffset + 1024
	unitTableBuffer, err := d2r.ReadRaw(baseAddress, 128*8)
	utils.IfError(err, "Failed to read unit table buffer")

	for i := 0; i < 128; i++ {
		offset := i * 8
		mobUnit := binary.LittleEndian.Uint64(unitTableBuffer[offset : offset+8])

		// log.Printf("mobUnit: %d", mobUnit)

		for mobUnit > 0 {
			// Read the mob structure data
			mobStructData, err := d2r.ReadRaw(uintptr(mobUnit), 144)
			utils.IfError(err, "Failed to read mob structure data")

			mobType, err := utils.ReadBufferAndAssert[uint32](mobStructData, 0x00, "UInt")
			utils.IfError(err, "Failed to read mobType")
			txtFileNo, err := utils.ReadBufferAndAssert[uint32](mobStructData, 0x04, "UInt")
			utils.IfError(err, "Failed to read txtFileNo")

			if !HideNPC(txtFileNo) {
				unitId, err := utils.ReadBufferAndAssert[uint32](mobStructData, 0x08, "UInt")
				utils.IfError(err, "Failed to read unitId")
				mode, err := utils.ReadBufferAndAssert[uint32](mobStructData, 0x0C, "UInt")
				utils.IfError(err, "Failed to read mode")
				pUnitData := binary.LittleEndian.Uint64(mobStructData[0x10:0x18])
				pPath := binary.LittleEndian.Uint64(mobStructData[0x38:0x40])

				dwOwnerId, err := utils.ReadAndAssert[uint32](d2r, uintptr(pUnitData+0x0C), "UInt")
				utils.IfError(err, "Failed to read dwOwnerId")
				isUnique, err := utils.ReadAndAssert[uint16](d2r, uintptr(pUnitData+0x18), "UShort")
				utils.IfError(err, "Failed to read isUnique")
				monsterFlag, err := utils.ReadAndAssert[uint8](d2r, uintptr(pUnitData+0x1A), "UChar")
				utils.IfError(err, "Failed to read monsterFlag")

				// Read data from pPath
				pathStructData, err := d2r.ReadRaw(uintptr(pPath), 16)
				utils.IfError(err, "Failed to read path structure data")

				monx, err := utils.ReadBufferAndAssert[uint16](pathStructData, 0x02, "UShort")
				utils.IfError(err, "Failed to read monx")
				mony, err := utils.ReadBufferAndAssert[uint16](pathStructData, 0x06, "UShort")
				utils.IfError(err, "Failed to read mony")
				xPosOffset, err := utils.ReadBufferAndAssert[uint16](pathStructData, 0x00, "UShort")
				utils.IfError(err, "Failed to read xPosOffset")
				yPosOffset, err := utils.ReadBufferAndAssert[uint16](pathStructData, 0x04, "UShort")
				utils.IfError(err, "Failed to read yPosOffset")

				// Calculate positions
				xPosOffsetFloat := float64(xPosOffset) / 65536.0
				yPosOffsetFloat := float64(yPosOffset) / 65536.0
				monxFloat := float64(monx) + xPosOffsetFloat
				monyFloat := float64(mony) + yPosOffsetFloat

				isHovered := false

				textTitle := getBossName(txtFileNo)
				isBoss := textTitle != ""

				// Get immunities and other stats
				pStatsListEx, err := utils.ReadBufferAndAssert[int64](mobStructData, 0x88, "Int64")
				utils.IfError(err, "Failed to read monsterFlag")

				statPtr, err := utils.ReadAndAssert[int64](d2r, uintptr(pStatsListEx+0x30), "Int64")
				utils.IfError(err, "Failed to read statPtr")
				statCount, err := utils.ReadAndAssert[int64](d2r, uintptr(pStatsListEx+0x38), "Int64")
				utils.IfError(err, "Failed to read statCount")

				isPlayerMinion := false
				playerMinion := getPlayerMinion(txtFileNo)
				if playerMinion != "" {
					isPlayerMinion = true
				} else {
					// Check if it's a revive
					value, err := utils.ReadAndAssert[uint32](d2r, uintptr(pStatsListEx+0xAC8+0xC), "UInt")
					if err == nil {
						isPlayerMinion = (value & 31) == 1
					}
				}

				isTownNPC := isTownNPC(txtFileNo)
				hp := uint32(0)
				maxhp := uint32(0)
				immunities := utils.Immunities{}

				if !isPlayerMinion {
					// Read stats
					statBufferSize := uintptr(statCount * 8)
					statsBuffer, err := d2r.ReadRaw(uintptr(statPtr)+0x2, uint32(statBufferSize))
					utils.IfError(err, "Failed to read stats buffer")

					for i := int64(0); i < statCount; i++ {
						offset := i * 8
						statEnum, err := utils.ReadBufferAndAssert[uint16](statsBuffer, int(offset), "UShort")
						utils.IfError(err, "Failed to read statEnum")
						statValue, err := utils.ReadBufferAndAssert[uint32](statsBuffer, int(offset+2), "UInt")
						utils.IfError(err, "Failed to read statValue")

						switch statEnum {
						case 36:
							immunities.Physical = statValue
						case 37:
							immunities.Magic = statValue
						case 39:
							immunities.Fire = statValue
						case 41:
							immunities.Light = statValue
						case 43:
							immunities.Cold = statValue
						case 45:
							immunities.Poison = statValue
						case 6:
							hp = statValue >> 8
						case 7:
							maxhp = statValue >> 8
						}
					}

					if currentHoveringUnitId != 0 && currentHoveringUnitId == unitId && isTownNPC == "" {
						isHovered = true
					}
				}

				mob := utils.Mob{
					TxtFileNo:      txtFileNo,
					Mode:           mode,
					Pos:            utils.UnitPosition{X: monxFloat, Y: monyFloat},
					IsUnique:       isUnique,
					IsBoss:         isBoss,
					MonsterFlag:    monsterFlag,
					IsPlayerMinion: isPlayerMinion,
					TextTitle:      textTitle,
					Immunities:     immunities,
					HP:             hp,
					MaxHP:          maxhp,
					IsTownNPC:      isTownNPC,
					IsHovered:      isHovered,
					DwOwnerId:      dwOwnerId,
					MobType:        mobType,
				}

				if isHovered {
					globals.HoveredMob = mob
				}

				globals.Mobs = append(globals.Mobs, mob)
			}

			// Get next mobUnit
			nextMobUnit, err := utils.ReadAndAssert[int64](d2r, uintptr(mobUnit+0x150), "Int64")
			utils.IfError(err, "Failed to read next mobUnit")
			if nextMobUnit == int64(mobUnit) || nextMobUnit == 0 {
				break
			}
			mobUnit = uint64(nextMobUnit)
		}
	}
}

func getBossName(txtFileNo uint32) string {
	switch txtFileNo {
	case 156:
		return "Andariel"
	case 211:
		return "Duriel"
	case 229:
		return "Radament"
	case 242:
		return "Mephisto"
	case 243:
		return "Diablo"
	case 250:
		return "Summoner"
	case 256:
		return "Izual"
	case 267:
		return "Bloodraven"
	case 333:
		return "Diabloclone"
	case 365:
		return "Griswold"
	case 526:
		return "Nihlathak"
	case 544:
		return "Baal"
	case 570:
		return "Baalclone"
	case 704:
		return "Uber Mephisto"
	case 705:
		return "Uber Diablo"
	case 706:
		return "Uber Izual"
	case 707:
		return "Uber Andariel"
	case 708:
		return "Uber Duriel"
	case 709:
		return "Uber Baal"
	default:
		return ""
	}
}

func getPlayerMinion(txtFileNo uint32) string {
	switch txtFileNo {
	case 271:
		return "roguehire"
	case 338:
		return "act2hire"
	case 359:
		return "act3hire"
	case 560:
		return "act5hire1"
	case 561:
		return "act5hire2"
	case 289:
		return "ClayGolem"
	case 290:
		return "BloodGolem"
	case 291:
		return "IronGolem"
	case 292:
		return "FireGolem"
	case 363:
		return "NecroSkeleton"
	case 364:
		return "NecroMage"
	case 417:
		return "ShadowWarrior"
	case 418:
		return "ShadowMaster"
	case 419:
		return "DruidHawk"
	case 420:
		return "DruidSpiritWolf"
	case 421:
		return "DruidFenris"
	case 423:
		return "HeartOfWolverine"
	case 424:
		return "OakSage"
	case 428:
		return "DruidBear"
	case 357:
		return "Valkyrie"
	default:
		return ""
	}
}

func GetSuperUniqueName(txtFileNo uint32) string {
	switch txtFileNo {
	case 0:
		return "Bonebreak"
	case 5:
		return "Corpsefire"
	case 11:
		return "Pitspawn Fouldog"
	case 20:
		return "Rakanishu"
	case 24:
		return "Treehead WoodFist"
	case 31:
		return "Fire Eye"
	case 45:
		return "The Countess"
	case 47:
		return "Sarina the Battlemaid"
	case 62:
		return "Baal Subject 1"
	case 66:
		return "Flamespike the Crawler"
	case 75:
		return "Fangskin"
	case 83:
		return "Bloodwitch the Wild"
	case 92:
		return "Beetleburst"
	case 97:
		return "Leatherarm"
	case 103:
		return "Ancient Kaa the Soulless"
	case 105:
		return "Baal Subject 2"
	case 120:
		return "The Tormentor"
	case 125:
		return "Web Mage the Burning"
	case 129:
		return "Stormtree"
	case 138:
		return "Icehawk Riftwing"
	case 160:
		return "Coldcrow"
	case 276:
		return "Boneash"
	case 281:
		return "Witch Doctor Endugu"
	case 284:
		return "Coldworm the Burrower"
	case 299:
		return "Taintbreeder"
	case 306:
		return "Grand Vizier of Chaos"
	case 308:
		return "Riftwraith the Cannibal"
	case 312:
		return "Lord De Seis"
	case 345:
		return "Council Member"
	case 346:
		return "Council Member"
	case 347:
		return "Council Member"
	case 362:
		return "Winged Death"
	case 402:
		return "The Smith"
	case 409:
		return "The Feature Creep"
	case 437:
		return "Bonesaw Breaker"
	case 440:
		return "Pindleskin"
	case 443:
		return "Threash Socket"
	case 449:
		return "Frozenstein"
	case 453:
		return "Megaflow Rectifier"
	case 472:
		return "Anodized Elite"
	case 475:
		return "Vinvear Molech"
	case 479:
		return "Siege Boss"
	case 481:
		return "Sharp Tooth Sayer"
	case 494:
		return "Dac Farren"
	case 496:
		return "Magma Torquer"
	case 501:
		return "Snapchip Shatter"
	case 508:
		return "Axe Dweller"
	case 529:
		return "Eyeback Unleashed"
	case 533:
		return "Blaze Ripper"
	case 540:
		return "Ancient Barbarian 1"
	case 541:
		return "Ancient Barbarian 2"
	case 542:
		return "Ancient Barbarian 3"
	case 557:
		return "Baal Subject 3"
	case 558:
		return "Baal Subject 4"
	case 571:
		return "Baal Subject 5"
	case 735:
		return "The Cow King"
	case 736:
		return "Dark Elder"
	default:
		return ""
	}
}

func isTownNPC(txtFileNo uint32) string {
	switch txtFileNo {
	case 146:
		return "DeckardCain"
	case 154:
		return "Charsi"
	case 147:
		return "Gheed"
	case 150:
		return "Kashya"
	case 155:
		return "Warriv"
	case 148:
		return "Akara"
	case 244:
		return "DeckardCain"
	case 210:
		return "Meshif"
	case 175:
		return "Warriv"
	case 199:
		return "Elzix"
	case 198:
		return "Greiz"
	case 177:
		return "Drognan"
	case 178:
		return "Fara"
	case 201:
		return "Jerhyn"
	case 202:
		return "Lysander"
	case 176:
		return "Atma"
	case 200:
		return "Geglash"
	case 331:
		return "Kaelan"
	case 245:
		return "DeckardCain"
	case 264:
		return "Meshif"
	case 255:
		return "Ormus"
	case 252:
		return "Asheara"
	case 254:
		return "Alkor"
	case 253:
		return "Hratli"
	case 297:
		return "Natalya"
	case 246:
		return "DeckardCain"
	case 251:
		return "Tyrael"
	case 367:
		return "Tyrael"
	case 521:
		return "Tyrael"
	case 257:
		return "Halbu"
	case 405:
		return "Jamella"
	case 265:
		return "DeckardCain"
	case 520:
		return "DeckardCain"
	case 512:
		return "Drehya"
	case 527:
		return "Drehya"
	case 515:
		return "Qual-Kehk"
	case 513:
		return "Malah"
	case 511:
		return "Larzuk"
	case 514:
		return "Nihlathak Town"
	case 266:
		return "navi"
	case 408:
		return "Malachai"
	case 406:
		return "Izual"
	default:
		return ""
	}
}

func HideNPC(txtFileNo uint32) bool {
	switch txtFileNo {
	case 149, 151, 152, 153, 157, 158, 159, 195, 196, 197, 179, 185, 203, 204, 205, 227, 268, 269, 272, 283, 293, 294, 296, 318, 319, 320, 321, 322, 323, 324, 325, 326, 327, 328, 329, 330, 332, 339, 344, 351, 352, 353, 355, 366, 370, 377, 378, 392, 393, 401, 410, 411, 412, 414, 415, 416, 543, 567, 568, 569, 711:
		return true
	default:
		return false
	}
}
