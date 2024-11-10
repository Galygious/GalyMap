// utils/structs.go

package utils

type UnitPosition struct {
	X float64
	Y float64
}

type ObjectPosition struct {
	X uint16
	Y uint16
}

type Player struct {
	Name              string
	UnitId            uint32
	Area              uint32
	PartyId           uint16
	Plevel            uint16
	Pos               UnitPosition
	IsHostileToPlayer bool
	Player            int
	PlayerName        string
	IsCorpse          bool
}

// Mob represents a monster or non-player character in the game.
type Mob struct {
	TxtFileNo      uint32
	Mode           uint32
	Pos            UnitPosition
	IsUnique       uint16
	IsBoss         bool
	MonsterFlag    uint8
	IsPlayerMinion bool
	TextTitle      string
	Immunities     Immunities
	HP             uint32
	MaxHP          uint32
	IsTownNPC      string
	IsHovered      bool
	MobType        uint32
	DwOwnerId      uint32
}

// Immunities represents the various immunities a Mob can have.
type Immunities struct {
	Physical uint32
	Magic    uint32
	Fire     uint32
	Light    uint32
	Cold     uint32
	Poison   uint32
}

// Object represents an in-game object with various properties.
type Object struct {
	TxtFileNo    uint32
	Name         string
	Mode         uint32
	IsChest      bool
	ChestState   string
	IsPortal     bool
	IsRedPortal  bool
	OwnerName    string
	InteractType uint8
	IsShrine     bool
	ShrineType   string
	Pos          ObjectPosition
	LevelNo      int
	UnitID       uint32
	ShrineFlag   uint16
}

type ProcessInfo struct {
	PID     uint32
	ExeName string
	Title   string
}
