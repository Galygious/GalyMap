// ui/overlay.go
package ui

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"GalyMap/config"
	"GalyMap/globals"
	"GalyMap/memory"
	"GalyMap/types"
	"GalyMap/utils"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"golang.org/x/sys/windows"
)

const (
	width          = 1920
	height         = 1080
	GWL_EXSTYLE    = uintptr(0xFFFFFFEC) // -20 in signed 32-bit converted to unsigned
	globalScale    = float64(4.6)
	overlayOffsetX = 2
	overlayOffsetY = -7

	// SetWindowPos flags
	HWND_TOPMOST   = ^uintptr(0) // (HWND)-1
	SWP_NOMOVE     = 0x0002
	SWP_NOSIZE     = 0x0001
	SWP_NOACTIVATE = 0x0010
)

var (
	spriteSheet          *image.RGBA
	spriteWidth          = 32
	spriteHeight         = 32
	sprites              []*Sprite
	program              uint32
	vao, vbo, ebo        uint32
	translationUniform   int32
	scaleUniform         int32
	modUser32            = windows.NewLazySystemDLL("user32.dll")
	procSetWindowLongPtr = modUser32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtr = modUser32.NewProc("GetWindowLongPtrW")
	procSetLayeredWindow = modUser32.NewProc("SetLayeredWindowAttributes")
	procSetWindowPos     = modUser32.NewProc("SetWindowPos")

	vertexShaderSource = `#version 410
in vec2 position;
uniform vec2 translation;
uniform vec2 scale;
void main() {
    gl_Position = vec4(position * scale + translation, 0.0, 1.0);
}` + "\x00"

	fragmentShaderSource = `#version 410
out vec4 color;
void main() {
    color = vec4(1.0, 0.0, 0.0, 1.0); // Red color
}` + "\x00"

	// Overlay Control
	overlayMutex  sync.Mutex
	overlayClosed bool

	// Synchronization with game data
	gameDataChan chan struct{}
)

// Sprite struct holds position and velocity
type Sprite struct {
	Position image.Point
	Velocity image.Point
}

func init() {
	// GLFW requires this to be called on the main thread
	runtime.LockOSThread()
}

// InitializeOverlay sets up the overlay window and OpenGL context
func InitializeOverlay(processInfo *globals.ProcessInfo, cfg *config.Settings) error {
	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize glfw: %v", err)
	}
	// Note: glfw.Terminate() will be called in RunOverlay when the overlay is closed

	// Configure GLFW window
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Decorated, glfw.False)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	// Create GLFW window
	window, err := glfw.CreateWindow(width, height, "Overlay", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}
	window.MakeContextCurrent()
	window.SetPos(0, 0) // Initial position; will be aligned with game window

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL: %v", err)
	}

	// Apply transparency, click-through styles, and set as topmost
	setTransparentAndClickThrough(window)

	// Load sprites from sprite sheet
	loadSprites("sprite_sheet.png")

	// Compile and link shaders
	program, err = newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		return fmt.Errorf("failed to create shader program: %v", err)
	}
	gl.UseProgram(program)

	// Retrieve uniform locations
	translationUniform = gl.GetUniformLocation(program, gl.Str("translation\x00"))
	scaleUniform = gl.GetUniformLocation(program, gl.Str("scale\x00"))

	// Define vertex data for a rectangle
	var vertices = []float32{
		-0.5, -0.5, // Bottom-left
		0.5, -0.5, // Bottom-right
		0.5, 0.5, // Top-right
		-0.5, 0.5, // Top-left
	}

	var indices = []uint32{
		0, 1, 2, // First triangle
		2, 3, 0, // Second triangle
	}

	// Generate and bind Vertex Array Object
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Generate and bind Vertex Buffer Object
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	// Generate and bind Element Buffer Object
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(indices), gl.Ptr(indices), gl.STATIC_DRAW)

	// Enable vertex attributes
	posAttrib := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(posAttrib)
	gl.VertexAttribPointer(posAttrib, 2, gl.FLOAT, false, 0, gl.PtrOffset(0))

	// Unbind VAO (optional)
	gl.BindVertexArray(0)

	// Initialize synchronization channel
	gameDataChan = make(chan struct{}, 1)

	return nil
}

// RunOverlay starts the overlay's main render loop
func RunOverlay() error {
	window := glfw.GetCurrentContext()
	if window == nil {
		return fmt.Errorf("glfw.GetCurrentContext() returned nil")
	}

	// Initialize OpenGL settings
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// // Start a goroutine to listen for overlay updates
	// go func() {
	// 	for range gameDataChan {
	// 		// Perform necessary updates
	// 		// This could involve setting flags, fetching new data, etc.
	// 		// For example: Update sprite positions based on new game data
	// 	}
	// }()

	// Main render loop
	for !window.ShouldClose() && !overlayClosed {
		// Clear the screen with transparent background
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Render all sprites based on current game data
		renderSprites()

		// Swap buffers and poll events
		window.SwapBuffers()
		glfw.PollEvents()
		time.Sleep(time.Millisecond * 6) // ~166 FPS
	}

	// Cleanup
	gl.DeleteProgram(program)
	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteBuffers(1, &ebo)

	glfw.Terminate()
	log.Println("Overlay window closed.")
	return nil
}

// CloseOverlay signals the overlay to close gracefully
func CloseOverlay() {
	overlayMutex.Lock()
	defer overlayMutex.Unlock()
	overlayClosed = true
}

// setTransparentAndClickThrough applies transparency, click-through styles, and sets the window as topmost
func setTransparentAndClickThrough(window *glfw.Window) {
	hwnd := uintptr(unsafe.Pointer(window.GetWin32Window()))

	// Get the current extended window style
	style, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(GWL_EXSTYLE))

	// Modify the style to include WS_EX_LAYERED and WS_EX_TRANSPARENT
	newStyle := style | 0x00080000 | 0x00000020 // WS_EX_LAYERED | WS_EX_TRANSPARENT
	procSetWindowLongPtr.Call(hwnd, uintptr(GWL_EXSTYLE), newStyle)

	// Set window transparency level (200 out of 255 for ~80% opacity)
	procSetLayeredWindow.Call(hwnd, 0, 200, 0x2) // LWA_ALPHA = 0x2

	// Set the window as topmost using SetWindowPos
	ret, _, err := procSetWindowPos.Call(
		hwnd,
		HWND_TOPMOST,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE,
	)
	if ret == 0 {
		log.Fatalf("SetWindowPos failed: %v", err)
	}
}

// loadSprites loads the sprite sheet and creates individual sprites
func loadSprites(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open sprite sheet: %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		log.Fatalf("Failed to decode sprite sheet: %v", err)
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}
	spriteSheet = rgba

	// Initialize sprites; this will be replaced with actual game data
	sprites = []*Sprite{
		{Position: image.Point{50, 50}, Velocity: image.Point{2, 0}},
		{Position: image.Point{100, 100}, Velocity: image.Point{0, 2}},
		{Position: image.Point{150, 150}, Velocity: image.Point{-1, -1}},
	}
}

// Update renderSprites to properly handle the player-centered view
func renderSprites() {
	// fmt.Printf("Rendering sprites\n")
	// Get ui
	uiOpen, err := utils.IsUIOpen()
	if err != nil {
		fmt.Printf("Error retrieving UI state: %v\n", err)
		return
	}

	if uiOpen {
		return
	}

	// Clear existing sprites
	sprites = make([]*Sprite, 0)

	// Get player position
	playerPos, err := utils.GetPlayerPosition()
	if err != nil {
		fmt.Printf("Error retrieving player position: %v\n", err)
		return
	}

	// Get mobs
	mobs, err := utils.GetMobs()
	if err != nil {
		fmt.Printf("Error retrieving mobs: %v\n", err)
		return
	}

	// Add mobs relative to player position
	for _, mob := range mobs {
		// Only render mobs within visible range
		if isWithinVisibleRange(mob.Pos.X, mob.Pos.Y, playerPos) {
			if !mob.IsCorpse && mob.HP > 0 {
				sprites = append(sprites, &Sprite{
					Position: image.Point{
						X: int(mob.Pos.X),
						Y: int(mob.Pos.Y),
					},
					Velocity: image.Point{X: 0, Y: 0},
				})
			}
		}
	}

	// Get Items
	items, err := utils.GetItems()
	if err != nil {
		fmt.Printf("Error retrieving items: %v\n", err)
		return
	}

	FilteredItems := globals.GetFilteredItems()
	GameMemoryData := globals.GetGameMemoryData()
	UnfilteredItems := make([]types.Item, 0)

	for _, item := range items {
		if item.ItemLoc == 3 || item.ItemLoc == 5 {
			filtered := false
			//check if any filtered items match this item
			// fmt.Printf("Checking item: %v\n", item.Name)
			for _, fp := range FilteredItems {
				if fp.Match(GameMemoryData["levelNo"].(uint32), item) {
					filtered = true
					continue
				}
			}
			if !filtered {
				UnfilteredItems = append(UnfilteredItems, item)
				fmt.Printf("Adding item to Unfiltered items: %v\n", item.Name)
			}
		}
	}

	displayadd := make([]types.ItemFootprint, 0)
	filteredadd := make([]types.ItemFootprint, 0)

	for _, item := range UnfilteredItems {
		if item.Filter() {
			fmt.Printf("Item matches a nip filter rule: %v\n", item.Name)
			// Item matches a nip filter rule add to displayed items
			displayadd = append(displayadd, types.ItemFootprint{
				Area:     GameMemoryData["levelNo"].(uint32),
				Position: image.Point{X: item.ItemX, Y: item.ItemY},
				Name:     item.Name,
				Quality:  item.QualityNo,
			})
		} else {
			fmt.Printf("Item does not match a nip filter rule: %v\n", item.Name)
			// Item does not match a nip filter
		}
		filteredadd = append(filteredadd, types.ItemFootprint{
			Area:     GameMemoryData["levelNo"].(uint32),
			Position: image.Point{X: item.ItemX, Y: item.ItemY},
			Name:     item.Name,
			Quality:  item.QualityNo,
		})
	}

	DisplayedItems := globals.GetDisplayedItems()
	DisplayedItems = append(DisplayedItems, displayadd...)
	updatedDisplayedItems := make([]types.ItemFootprint, 0)

	for _, displayedItem := range DisplayedItems {
		found := false
		for _, item := range items {
			if displayedItem.Match(GameMemoryData["levelNo"].(uint32), item) {
				found = true
				break
			}
		}
		if found {
			updatedDisplayedItems = append(updatedDisplayedItems, displayedItem)
		} else {
			// Item no longer exists in game
			fmt.Printf("Item no longer exists in game: %v\n", displayedItem.Name)
		}
	}

	FilteredItems = append(FilteredItems, filteredadd...)
	globals.SetDisplayedItems(updatedDisplayedItems)
	globals.SetFilteredItems(FilteredItems)

	// Add Display Items
	for _, item := range DisplayedItems {
		sprites = append(sprites, &Sprite{
			Position: item.Position,
			Velocity: image.Point{X: 0, Y: 0},
		})
	}

	// Add player sprite at the center
	sprites = append(sprites, &Sprite{
		Position: image.Point{
			X: int(playerPos.X),
			Y: int(playerPos.Y),
		},
		Velocity: image.Point{X: 0, Y: 0},
	})

	// Render all sprites
	for _, sprite := range sprites {
		renderSprite(sprite, playerPos)
	}
}

// renderSprite draws a single sprite at its position using shaders
func renderSprite(sprite *Sprite, playerPos globals.UnitPosition) {
	gl.UseProgram(program)

	// Convert sprite position from game coordinates to Normalized Device Coordinates (NDC)
	x_ndc, y_ndc := gameToScreenCoordinatesFloat(float64(sprite.Position.X), float64(sprite.Position.Y), playerPos)

	// Calculate scale in NDC based on sprite size
	scaleX := float32(spriteWidth) / float32(width)
	scaleY := float32(spriteHeight) / float32(height)

	// Set uniform values for translation and scale
	gl.Uniform2f(translationUniform, x_ndc, y_ndc)
	gl.Uniform2f(scaleUniform, scaleX, scaleY)

	// Bind VAO and draw the rectangle
	gl.BindVertexArray(vao)
	gl.DrawElements(gl.TRIANGLES, int32(6), gl.UNSIGNED_INT, nil)
}

// Helper function to determine if a position is within visible range
func isWithinVisibleRange(x, y float64, playerPos globals.UnitPosition) bool {
	// Calculate distance from player
	dx := x - playerPos.X
	dy := y - playerPos.Y

	// Define visible range based on screen size and scale
	maxRange := float64(width) / (2 * globalScale) // Adjust this calculation based on your needs

	// Check if position is within visible range
	distanceSquared := dx*dx + dy*dy
	return distanceSquared <= maxRange*maxRange
}

// Transform game coordinates into screen coordinates while keeping the player centered
func gameToScreenCoordinates(gameX, gameY float64, playerPos globals.UnitPosition) (int, int) {
	// Calculate the relative position from player
	relativeX := gameX - playerPos.X
	relativeY := gameY - playerPos.Y

	// Define constants
	const (
		// Base scale factor (adjust this to change the overall zoom level)
		baseScale = globalScale
		// Isometric rotation angle (45 degrees)
		angleRadians = math.Pi / 4
		// Y-axis compression factor for isometric view
		yCompression = 0.5
	)

	// Scale can be adjusted dynamically if needed
	scale := baseScale

	// Apply isometric rotation
	rotatedX := relativeX*math.Cos(angleRadians) - relativeY*math.Sin(angleRadians)
	rotatedY := relativeX*math.Sin(angleRadians) + relativeY*math.Cos(angleRadians)

	// Apply scaling and Y compression
	scaledX := rotatedX * scale
	scaledY := rotatedY * scale * yCompression

	// Convert to screen coordinates (player is always at center)
	screenX := float64(width/2) + scaledX + float64(overlayOffsetX)
	screenY := float64(height/2) + scaledY + float64(overlayOffsetY)

	// Bounds checking to ensure coordinates stay within screen
	screenX = math.Max(0, math.Min(float64(width), screenX))
	screenY = math.Max(0, math.Min(float64(height), screenY))

	return int(screenX), int(screenY)
}

// Convert game coordinates to NDC (Normalized Device Coordinates)
func gameToScreenCoordinatesFloat(gameX, gameY float64, playerPos globals.UnitPosition) (float32, float32) {
	screenX, screenY := gameToScreenCoordinates(gameX, gameY, playerPos)

	// Convert to NDC space (-1 to 1)
	ndcX := (float32(screenX)/float32(width))*2.0 - 1.0
	ndcY := -((float32(screenY)/float32(height))*2.0 - 1.0)

	return ndcX, ndcY
}

// compileShader compiles a shader of a given type from source
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	defer free()
	gl.ShaderSource(shader, 1, csources, nil)
	gl.CompileShader(shader)

	// Check for compilation errors
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		logStr := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(logStr))

		return 0, fmt.Errorf("failed to compile shader: %v", logStr)
	}

	return shader, nil
}

// newProgram links vertex and fragment shaders into a shader program
func newProgram(vertexSrc, fragmentSrc string) (uint32, error) {
	vertexShader, err := compileShader(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := compileShader(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(fragmentShader)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLength)

		logStr := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(prog, logLength, nil, gl.Str(logStr))

		return 0, fmt.Errorf("failed to link program: %v", logStr)
	}

	return prog, nil
}

// ReadGameMemoryRoutine continuously reads game memory and updates globals
func ReadGameMemoryRoutine(d2r *utils.ClassMemory, cfg *config.Settings) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Check if player is in-game and log the result
	unitTableOffset, exists := globals.GetOffset("unitTable")
	if !exists {
		log.Fatalf("main.go: Failed to retrieve 'unitTable' offset from globals.")
	}
	inGame, err := memory.IsInGame(d2r, unitTableOffset)
	if err != nil {
		log.Fatalf("Failed to check if player is in-game: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			globals.IncrementTicktock()
			if inGame {
				memory.ReadGameMemory(d2r, globals.Settings)
			}
		}
	}
}
