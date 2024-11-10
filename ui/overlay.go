// ui/overlay.go

package ui

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"golang.org/x/sys/windows"
	"your_project/globals"
	"your_project/utils"
)

const (
	width       = 800
	height      = 600
	GWL_EXSTYLE = uintptr(0xFFFFFFEC) // -20 in signed 32-bit converted to unsigned

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
    color = vec4(1.0, 1.0, 1.0, 1.0); // White color
}` + "\x00"
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
func InitializeOverlay() error {
	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize glfw: %v", err)
	}
	// Note: Terminate will be called when the overlay is closed

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
	window.SetPos(100, 100) // Adjust position as needed

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

	return nil
}

// RunOverlay starts the overlay's main render loop
func RunOverlay() {
	// Find the overlay window
	window := glfw.GetCurrentContext()

	// Main render loop
	for !window.ShouldClose() {
		// Clear the screen with transparent background
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Render all sprites
		renderSprites()

		// Swap buffers and poll events
		window.SwapBuffers()
		glfw.PollEvents()
		time.Sleep(time.Millisecond * 16) // ~60 FPS
	}

	// Cleanup
	gl.DeleteProgram(program)
	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteBuffers(1, &ebo)
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

	// Initialize sprites with default positions and velocities
	// This will be replaced with actual game data
	sprites = []*Sprite{
		{Position: image.Point{50, 50}, Velocity: image.Point{2, 0}},
		{Position: image.Point{100, 100}, Velocity: image.Point{0, 2}},
		{Position: image.Point{150, 150}, Velocity: image.Point{-1, -1}},
	}
}

// renderSprites renders all sprites onto the transparent window
func renderSprites() {
	for _, sprite := range sprites {
		renderSprite(sprite)
		updatePosition(sprite)
	}
}

// renderSprite draws a single sprite at its position using shaders
func renderSprite(sprite *Sprite) {
	gl.UseProgram(program)

	// Convert sprite position from window coordinates to Normalized Device Coordinates (NDC)
	x_ndc := (float32(sprite.Position.X)/float32(width))*2.0 - 1.0
	y_ndc := -((float32(sprite.Position.Y)/float32(height))*2.0 - 1.0)

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

// updatePosition moves the sprite and wraps it within window bounds
func updatePosition(sprite *Sprite) {
	sprite.Position.X += sprite.Velocity.X
	sprite.Position.Y += sprite.Velocity.Y

	// Reverse direction if sprite goes out of bounds
	if sprite.Position.X < 0 || sprite.Position.X > width-spriteWidth {
		sprite.Velocity.X *= -1
	}
	if sprite.Position.Y < 0 || sprite.Position.Y > height-spriteHeight {
		sprite.Velocity.Y *= -1
	}
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
