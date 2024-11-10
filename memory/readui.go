package memory

import (
	"GalyMap/globals"
	"GalyMap/utils"
)

func ReadUI(d2r *utils.ClassMemory) (bool, error) {
	base := d2r.BaseAddress + globals.Offsets.M["uiOffset"] - 0xa
	buffer, err := d2r.ReadRaw(base, 32)
	if err != nil {
		return false, err
	}

	invMenu := buffer[0x01]
	charMenu := buffer[0x02]
	skillSelect := buffer[0x03]
	skillMenu := buffer[0x04]
	// npcInteract := buffer[0x08]
	quitMenu := buffer[0x09]
	// npcShop := buffer[0x0B]
	questsMenu := buffer[0x0E]
	waypointMenu := buffer[0x13]
	stash := buffer[0x18]
	partyMenu := buffer[0x15]
	mercMenu := buffer[0x1E]
	// loading := buffer[0x16C]

	leftMenu := questsMenu | charMenu | mercMenu | partyMenu | waypointMenu | stash
	rightMenu := skillMenu | invMenu

	UIShown := false
	if rightMenu != 0 || quitMenu != 0 {
		UIShown = true
	}
	if leftMenu != 0 || quitMenu != 0 || skillSelect != 0 {
		UIShown = true
	}
	return UIShown, nil
}
