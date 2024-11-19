// globals/globals.go

package globals

import (
	"GalyMap/types"
	"log"
	"sync"
	"sync/atomic"
)

// Define constants for location keys
const (
	Ground    = "Ground"
	Belt      = "Belt"
	Equipped  = "Equipped"
	Inventory = "Inventory"
	Socket    = "Socket"
)

var (
	Ticktock       int64
	Items          []types.Item
	Mobs           []Mob
	HoveredMob     Mob
	GameObjects    []Object
	OtherPlayers   []Player
	PartyList      []Player
	MapSeed        uint32
	ItemAlertList  map[string]bool
	GameMemoryData map[string]interface{}
	Settings       map[string]bool
	FilteredItems  []types.ItemFootprint
	DisplayedItems []types.ItemFootprint

	// Mutexes for synchronizing access to shared data
	GameDataMutex       sync.RWMutex
	OffsetsMutex        sync.RWMutex
	FilteredItemsMutex  sync.RWMutex
	DisplayedItemsMutex sync.RWMutex

	// Offsets for reading memory
	Offsets struct {
		M map[string]uintptr
	}

	Location = map[string][]int{
		Ground:    {3, 5},
		Belt:      {2},
		Equipped:  {1},
		Inventory: {0},
		Socket:    {6},
	}
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

	// Initialize GameMemoryData and ItemAlertList maps
	GameMemoryData = make(map[string]interface{})
	ItemAlertList = make(map[string]bool)
	FilteredItems = make([]types.ItemFootprint, 0)

	// Initialize Offsets map
	Offsets.M = make(map[string]uintptr)
	log.Println("Globals settings initialized.")
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
	GameDataMutex.RLock()
	defer GameDataMutex.RUnlock()

	for k, v := range data {
		GameMemoryData[k] = v
	}
}

// GetFilteredItems safely retrieves a copy of FilteredItems
func GetFilteredItems() []types.ItemFootprint {
	FilteredItemsMutex.RLock()
	defer FilteredItemsMutex.RUnlock()

	// Create a copy to prevent race conditions
	itemsCopy := make([]types.ItemFootprint, len(FilteredItems))
	copy(itemsCopy, FilteredItems)
	return itemsCopy
}

// SetFilteredItems safely updates FilteredItems
func SetFilteredItems(items []types.ItemFootprint) {
	FilteredItemsMutex.RLock()
	defer FilteredItemsMutex.RUnlock()
	FilteredItems = items
}

// GetDisplayedItems safely retrieves a copy of DisplayedItems
func GetDisplayedItems() []types.ItemFootprint {
	DisplayedItemsMutex.RLock()
	defer DisplayedItemsMutex.RUnlock()

	// create a copy to prevent race conditions
	itemsCopy := make([]types.ItemFootprint, len(DisplayedItems))
	copy(itemsCopy, DisplayedItems)
	return itemsCopy
}

// SetDisplayedItems safely updates DisplayedItems
func SetDisplayedItems(items []types.ItemFootprint) {
	DisplayedItemsMutex.RLock()
	defer DisplayedItemsMutex.RUnlock()
	DisplayedItems = items
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

	if Offsets.M == nil {
		log.Fatalf("Attempted to set offset '%s' but Offsets.M is nil. Ensure InitSettings() is called before using SetOffset.", key)
	}

	Offsets.M[key] = value
}
