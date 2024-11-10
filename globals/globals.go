// globals/globals.go
package globals

import (
	"GalyMap/types"
	"GalyMap/utils"
	"sync"
	"sync/atomic"
)

var (
	Ticktock       int64
	Items          []types.Item
	Mobs           []utils.Mob
	HoveredMob     utils.Mob
	GameObjects    []utils.Object
	OtherPlayers   []utils.Player
	PartyList      []utils.Player
	MapSeed        uint32
	ItemAlertList  map[string]bool
	GameMemoryData map[string]interface{}
	Settings       map[string]bool

	// Mutexes for synchronizing access to shared data
	GameDataMutex sync.RWMutex
	OffsetsMutex  sync.RWMutex
)

// Initialize the Settings map with default values
func InitSettings() {
	Settings = map[string]bool{
		"showOtherPlayers":   true,
		"showNormalMobs":     true,
		"showUniqueMobs":     true,
		"showBosses":         true,
		"showDeadMobs":       true,
		"showPlayerMissiles": true,
		"showEnemyMissiles":  true,
		"enableItemFilter":   true,
		"showShrines":        true,
		"showPortals":        true,
		"showChests":         true,
	}

	// Initialize GameMemoryData map
	GameMemoryData = make(map[string]interface{})
	ItemAlertList = make(map[string]bool)
}

// IncrementTicktock safely increments the tick counter
func IncrementTicktock() {
	atomic.AddInt64(&Ticktock, 1)
}

// Safe getters and setters for shared data

// GetGameMemoryData safely retrieves a copy of GameMemoryData
func GetGameMemoryData() map[string]interface{} {
	GameDataMutex.RLock()
	defer GameDataMutex.RUnlock()

	// Create a copy to prevent race conditions
	dataCopy := make(map[string]interface{})
	for k, v := range GameMemoryData {
		dataCopy[k] = v
	}
	return dataCopy
}

// SetGameMemoryData safely updates GameMemoryData
func SetGameMemoryData(data map[string]interface{}) {
	GameDataMutex.Lock()
	defer GameDataMutex.Unlock()

	for k, v := range data {
		GameMemoryData[k] = v
	}
}

// GetOffset safely retrieves an offset value by key
func GetOffset(key string) (uintptr, bool) {
	OffsetsMutex.RLock()
	defer OffsetsMutex.RUnlock()

	value, exists := Offsets.M[key]
	return value, exists
}

// SetOffset safely sets an offset value by key
func SetOffset(key string, value uintptr) {
	OffsetsMutex.Lock()
	defer OffsetsMutex.Unlock()

	Offsets.M[key] = value
}
