// Package graphics provides comprehensive 2D and 3D graphics capabilities.
// This package includes advanced drawing primitives, image manipulation,
// 3D rendering, shaders, and animation frameworks.
package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"sync"
)

// Point represents a 2D point.
type Point struct {
	X, Y float64
}

// Point3D represents a 3D point.
type Point3D struct {
	X, Y, Z float64
}

// Color represents an RGBA color.
type Color struct {
	R, G, B, A uint8
}

// Canvas represents a drawing canvas.
type Canvas struct {
	Width, Height int
	Image         *image.RGBA
	Context       *Context2D
}

// Context2D provides 2D drawing operations.
type Context2D struct {
	canvas      *Canvas
	fillColor   Color
	strokeColor Color
	lineWidth   float64
	font        Font
}

// Font represents a font configuration.
type Font struct {
	Family string
	Size   float64
	Style  FontStyle
	Weight FontWeight
}

// FontStyle represents font styles.
type FontStyle int

const (
	FontStyleNormal FontStyle = iota
	FontStyleItalic
	FontStyleOblique
)

// FontWeight represents font weights.
type FontWeight int

const (
	FontWeightThin FontWeight = iota
	FontWeightLight
	FontWeightNormal
	FontWeightMedium
	FontWeightBold
	FontWeightExtraBold
	FontWeightBlack
)

// Matrix3x3 represents a 3x3 transformation matrix.
type Matrix3x3 struct {
	M00, M01, M02 float64
	M10, M11, M12 float64
	M20, M21, M22 float64
}

// Vector2 represents a 2D vector.
type Vector2 struct {
	X, Y float64
}

// Vector3 represents a 3D vector.
type Vector3 struct {
	X, Y, Z float64
}

// Vector4 represents a 4D vector.
type Vector4 struct {
	X, Y, Z, W float64
}

// Matrix4x4 represents a 4x4 transformation matrix.
type Matrix4x4 struct {
	M [16]float64
}

// Vertex represents a 3D vertex.
type Vertex struct {
	Position Vector3
	Normal   Vector3
	UV       Vector2
	Color    Color
}

// Triangle represents a 3D triangle.
type Triangle struct {
	V0, V1, V2 Vertex
}

// Mesh represents a 3D mesh.
type Mesh struct {
	Vertices []Vertex
	Indices  []uint32
	Material Material
}

// Material represents surface material properties.
type Material struct {
	DiffuseColor  Color
	SpecularColor Color
	AmbientColor  Color
	Shininess     float64
	Opacity       float64
	Texture       *Texture
}

// Texture represents a texture.
type Texture struct {
	Width, Height int
	Data          []Color
	Filtering     TextureFiltering
	Wrapping      TextureWrapping
}

// TextureFiltering represents texture filtering modes.
type TextureFiltering int

const (
	FilterNearest TextureFiltering = iota
	FilterLinear
	FilterBilinear
	FilterTrilinear
)

// TextureWrapping represents texture wrapping modes.
type TextureWrapping int

const (
	WrapRepeat TextureWrapping = iota
	WrapClamp
	WrapMirror
)

// Camera represents a 3D camera.
type Camera struct {
	Position Vector3
	Target   Vector3
	Up       Vector3
	FOV      float64
	Aspect   float64
	Near     float64
	Far      float64
}

// Light represents a light source.
type Light struct {
	Type        LightType
	Position    Vector3
	Direction   Vector3
	Color       Color
	Intensity   float64
	Attenuation Vector3 // constant, linear, quadratic
	SpotAngle   float64
}

// LightType represents different light types.
type LightType int

const (
	DirectionalLight LightType = iota
	PointLight
	SpotLight
	AmbientLight
)

// Scene3D represents a 3D scene.
type Scene3D struct {
	Meshes   []Mesh
	Lights   []Light
	Camera   Camera
	SkyColor Color
}

// Renderer3D provides 3D rendering capabilities.
type Renderer3D struct {
	Width, Height int
	ColorBuffer   []Color
	DepthBuffer   []float64
	ViewMatrix    Matrix4x4
	ProjectMatrix Matrix4x4
	Shading       ShadingMode
	Culling       bool
	mutex         sync.Mutex
}

// ShadingMode represents different shading modes.
type ShadingMode int

const (
	FlatShading ShadingMode = iota
	GouraudShading
	PhongShading
)

// Animation represents an animation.
type Animation struct {
	Name      string
	Duration  float64
	Keyframes []Keyframe
	Loop      bool
	Playing   bool
	Time      float64
}

// Keyframe represents an animation keyframe.
type Keyframe struct {
	Time     float64
	Position Vector3
	Rotation Vector4 // Quaternion
	Scale    Vector3
}

// Animator manages animations.
type Animator struct {
	Animations  map[string]*Animation
	CurrentAnim string
}

// Particle represents a particle.
type Particle struct {
	Position Vector3
	Velocity Vector3
	Color    Color
	Size     float64
	Life     float64
	MaxLife  float64
}

// ParticleSystem manages particle effects.
type ParticleSystem struct {
	Particles       []Particle
	MaxParticles    int
	EmissionRate    float64
	Gravity         Vector3
	StartColor      Color
	EndColor        Color
	StartSize       float64
	EndSize         float64
	ParticleLife    float64
	EmitterPosition Vector3
}

// Shader represents a graphics shader.
type Shader struct {
	VertexShader   string
	FragmentShader string
	Uniforms       map[string]interface{}
}

// RenderPass represents a rendering pass.
type RenderPass struct {
	Name       string
	Shader     Shader
	RenderFunc func(*Renderer3D, *Scene3D)
}

// PostProcessor handles post-processing effects.
type PostProcessor struct {
	Passes      []RenderPass
	TempBuffers [][]Color
}

// FontStyle represents font styling options.
type FontStyle int

const (
	FontNormal FontStyle = iota
	FontBold
	FontItalic
	FontBoldItalic
)

// NewCanvas creates a new drawing canvas.
func NewCanvas(width, height int) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	canvas := &Canvas{
		Width:  width,
		Height: height,
		Image:  img,
	}

	canvas.Context = &Context2D{
		canvas:      canvas,
		fillColor:   Color{0, 0, 0, 255},
		strokeColor: Color{0, 0, 0, 255},
		lineWidth:   1.0,
		font:        Font{Family: "Arial", Size: 12, Style: FontNormal},
	}

	// Fill with white background
	canvas.Context.SetFillColor(Color{255, 255, 255, 255})
	canvas.Context.FillRect(0, 0, float64(width), float64(height))
	canvas.Context.SetFillColor(Color{0, 0, 0, 255})

	return canvas
}

// LoadImage loads an image from a file.
func LoadImage(filename string) (*Canvas, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	canvas := &Canvas{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Image:  rgba,
	}

	canvas.Context = &Context2D{
		canvas:      canvas,
		fillColor:   Color{0, 0, 0, 255},
		strokeColor: Color{0, 0, 0, 255},
		lineWidth:   1.0,
		font:        Font{Family: "Arial", Size: 12, Style: FontNormal},
	}

	return canvas, nil
}

// SavePNG saves the canvas as a PNG file.
func (c *Canvas) SavePNG(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, c.Image)
}

// SaveJPEG saves the canvas as a JPEG file.
func (c *Canvas) SaveJPEG(filename string, quality int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	options := &jpeg.Options{Quality: quality}
	return jpeg.Encode(file, c.Image, options)
}

// Context2D methods

// SetFillColor sets the fill color.
func (ctx *Context2D) SetFillColor(c Color) {
	ctx.fillColor = c
}

// SetStrokeColor sets the stroke color.
func (ctx *Context2D) SetStrokeColor(c Color) {
	ctx.strokeColor = c
}

// SetLineWidth sets the line width.
func (ctx *Context2D) SetLineWidth(width float64) {
	ctx.lineWidth = width
}

// SetFont sets the font.
func (ctx *Context2D) SetFont(font Font) {
	ctx.font = font
}

// Clear clears the canvas with a color.
func (ctx *Context2D) Clear(c Color) {
	ctx.FillRect(0, 0, float64(ctx.canvas.Width), float64(ctx.canvas.Height))
}

// FillRect fills a rectangle.
func (ctx *Context2D) FillRect(x, y, width, height float64) {
	startX := int(x)
	startY := int(y)
	endX := int(x + width)
	endY := int(y + height)

	for py := startY; py < endY && py < ctx.canvas.Height; py++ {
		for px := startX; px < endX && px < ctx.canvas.Width; px++ {
			if px >= 0 && py >= 0 {
				ctx.canvas.Image.Set(px, py, ctx.colorToRGBA(ctx.fillColor))
			}
		}
	}
}

// StrokeRect strokes a rectangle outline.
func (ctx *Context2D) StrokeRect(x, y, width, height float64) {
	// Top edge
	ctx.DrawLine(x, y, x+width, y)
	// Right edge
	ctx.DrawLine(x+width, y, x+width, y+height)
	// Bottom edge
	ctx.DrawLine(x+width, y+height, x, y+height)
	// Left edge
	ctx.DrawLine(x, y+height, x, y)
}

// FillCircle fills a circle.
func (ctx *Context2D) FillCircle(centerX, centerY, radius float64) {
	radiusInt := int(radius)
	centerXInt := int(centerX)
	centerYInt := int(centerY)

	for y := -radiusInt; y <= radiusInt; y++ {
		for x := -radiusInt; x <= radiusInt; x++ {
			if x*x+y*y <= radiusInt*radiusInt {
				px := centerXInt + x
				py := centerYInt + y
				if px >= 0 && py >= 0 && px < ctx.canvas.Width && py < ctx.canvas.Height {
					ctx.canvas.Image.Set(px, py, ctx.colorToRGBA(ctx.fillColor))
				}
			}
		}
	}
}

// StrokeCircle strokes a circle outline.
func (ctx *Context2D) StrokeCircle(centerX, centerY, radius float64) {
	// Bresenham's circle algorithm
	x := int(radius)
	y := 0
	radiusError := 1 - x

	centerXInt := int(centerX)
	centerYInt := int(centerY)

	for x >= y {
		ctx.setPixel(centerXInt+x, centerYInt+y, ctx.strokeColor)
		ctx.setPixel(centerXInt+y, centerYInt+x, ctx.strokeColor)
		ctx.setPixel(centerXInt-x, centerYInt+y, ctx.strokeColor)
		ctx.setPixel(centerXInt-y, centerYInt+x, ctx.strokeColor)
		ctx.setPixel(centerXInt-x, centerYInt-y, ctx.strokeColor)
		ctx.setPixel(centerXInt-y, centerYInt-x, ctx.strokeColor)
		ctx.setPixel(centerXInt+x, centerYInt-y, ctx.strokeColor)
		ctx.setPixel(centerXInt+y, centerYInt-x, ctx.strokeColor)

		y++
		if radiusError < 0 {
			radiusError += 2*y + 1
		} else {
			x--
			radiusError += 2*(y-x) + 1
		}
	}
}

// DrawLine draws a line between two points.
func (ctx *Context2D) DrawLine(x1, y1, x2, y2 float64) {
	// Bresenham's line algorithm
	dx := math.Abs(x2 - x1)
	dy := math.Abs(y2 - y1)

	var sx, sy int
	if x1 < x2 {
		sx = 1
	} else {
		sx = -1
	}
	if y1 < y2 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy
	x, y := int(x1), int(y1)

	for {
		ctx.setPixel(x, y, ctx.strokeColor)

		if x == int(x2) && y == int(y2) {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// DrawPolygon draws a polygon.
func (ctx *Context2D) DrawPolygon(points []Point) {
	if len(points) < 2 {
		return
	}

	for i := 0; i < len(points); i++ {
		next := (i + 1) % len(points)
		ctx.DrawLine(points[i].X, points[i].Y, points[next].X, points[next].Y)
	}
}

// FillPolygon fills a polygon.
func (ctx *Context2D) FillPolygon(points []Point) {
	if len(points) < 3 {
		return
	}

	// Simple scanline fill algorithm
	minY := int(points[0].Y)
	maxY := int(points[0].Y)

	for _, p := range points {
		if int(p.Y) < minY {
			minY = int(p.Y)
		}
		if int(p.Y) > maxY {
			maxY = int(p.Y)
		}
	}

	for y := minY; y <= maxY; y++ {
		intersections := make([]int, 0)

		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			p1, p2 := points[i], points[j]

			if (int(p1.Y) <= y && y < int(p2.Y)) || (int(p2.Y) <= y && y < int(p1.Y)) {
				// Calculate intersection
				x := int(p1.X + (float64(y)-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y))
				intersections = append(intersections, x)
			}
		}

		// Sort intersections
		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[i] > intersections[j] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}

		// Fill between pairs of intersections
		for i := 0; i < len(intersections); i += 2 {
			if i+1 < len(intersections) {
				for x := intersections[i]; x <= intersections[i+1]; x++ {
					ctx.setPixel(x, y, ctx.fillColor)
				}
			}
		}
	}
}

// DrawText draws text (simplified implementation).
func (ctx *Context2D) DrawText(text string, x, y float64) {
	// This is a very basic text rendering
	// In a real implementation, you would use a proper font renderer
	for i, char := range text {
		ctx.drawChar(char, x+float64(i)*8, y)
	}
}

func (ctx *Context2D) drawChar(char rune, x, y float64) {
	// Very basic character rendering (just for demonstration)
	// This would typically use actual font data
	switch char {
	case 'A':
		ctx.DrawLine(x, y+8, x+4, y)
		ctx.DrawLine(x+4, y, x+8, y+8)
		ctx.DrawLine(x+2, y+4, x+6, y+4)
	case 'B':
		ctx.DrawLine(x, y, x, y+8)
		ctx.DrawLine(x, y, x+6, y)
		ctx.DrawLine(x, y+4, x+6, y+4)
		ctx.DrawLine(x, y+8, x+6, y+8)
		ctx.DrawLine(x+6, y, x+6, y+4)
		ctx.DrawLine(x+6, y+4, x+6, y+8)
	default:
		// Draw a simple rectangle for unknown characters
		ctx.StrokeRect(x, y, 8, 8)
	}
}

func (ctx *Context2D) setPixel(x, y int, c Color) {
	if x >= 0 && y >= 0 && x < ctx.canvas.Width && y < ctx.canvas.Height {
		ctx.canvas.Image.Set(x, y, ctx.colorToRGBA(c))
	}
}

func (ctx *Context2D) colorToRGBA(c Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// 3D Graphics support

// Matrix4 represents a 4x4 transformation matrix.
type Matrix4 [16]float64

// Vector3 represents a 3D vector.
type Vector3 struct {
	X, Y, Z float64
}

// Camera represents a 3D camera.
type Camera struct {
	Position Point3D
	Target   Point3D
	Up       Vector3
	FOV      float64
	Aspect   float64
	Near     float64
	Far      float64
}

// Mesh represents a 3D mesh.
type Mesh struct {
	Vertices []Point3D
	Faces    []Face
}

// Face represents a triangle face.
type Face struct {
	V1, V2, V3 int
}

// Renderer3D provides 3D rendering capabilities.
type Renderer3D struct {
	canvas   *Canvas
	camera   *Camera
	viewport Matrix4
	world    Matrix4
	view     Matrix4
	proj     Matrix4
}

// NewRenderer3D creates a new 3D renderer.
func NewRenderer3D(canvas *Canvas) *Renderer3D {
	camera := &Camera{
		Position: Point3D{0, 0, 5},
		Target:   Point3D{0, 0, 0},
		Up:       Vector3{0, 1, 0},
		FOV:      60,
		Aspect:   float64(canvas.Width) / float64(canvas.Height),
		Near:     0.1,
		Far:      100,
	}

	renderer := &Renderer3D{
		canvas: canvas,
		camera: camera,
	}

	renderer.updateMatrices()
	return renderer
}

// SetCamera sets the camera.
func (r *Renderer3D) SetCamera(camera *Camera) {
	r.camera = camera
	r.updateMatrices()
}

// RenderMesh renders a 3D mesh.
func (r *Renderer3D) RenderMesh(mesh *Mesh) {
	for _, face := range mesh.Faces {
		v1 := r.projectVertex(mesh.Vertices[face.V1])
		v2 := r.projectVertex(mesh.Vertices[face.V2])
		v3 := r.projectVertex(mesh.Vertices[face.V3])

		// Draw triangle edges
		r.canvas.Context.DrawLine(v1.X, v1.Y, v2.X, v2.Y)
		r.canvas.Context.DrawLine(v2.X, v2.Y, v3.X, v3.Y)
		r.canvas.Context.DrawLine(v3.X, v3.Y, v1.X, v1.Y)
	}
}

func (r *Renderer3D) projectVertex(vertex Point3D) Point {
	// Apply world, view, and projection transformations
	// This is a simplified projection

	// Calculate distance from camera
	dx := vertex.X - r.camera.Position.X
	dy := vertex.Y - r.camera.Position.Y
	dz := vertex.Z - r.camera.Position.Z

	// Simple perspective projection
	if dz <= 0 {
		dz = 0.1 // Avoid division by zero
	}

	scale := 200.0 / dz // Perspective scaling factor

	screenX := float64(r.canvas.Width)/2 + dx*scale
	screenY := float64(r.canvas.Height)/2 - dy*scale

	return Point{X: screenX, Y: screenY}
}

func (r *Renderer3D) updateMatrices() {
	// Update view and projection matrices
	// This is simplified - in a real implementation you'd use proper matrix math
}

// Utility functions for creating common shapes

// CreateCube creates a cube mesh.
func CreateCube(size float64) *Mesh {
	half := size / 2

	vertices := []Point3D{
		{-half, -half, -half}, // 0
		{half, -half, -half},  // 1
		{half, half, -half},   // 2
		{-half, half, -half},  // 3
		{-half, -half, half},  // 4
		{half, -half, half},   // 5
		{half, half, half},    // 6
		{-half, half, half},   // 7
	}

	faces := []Face{
		// Front face
		{0, 1, 2}, {0, 2, 3},
		// Back face
		{4, 7, 6}, {4, 6, 5},
		// Left face
		{0, 3, 7}, {0, 7, 4},
		// Right face
		{1, 5, 6}, {1, 6, 2},
		// Top face
		{3, 2, 6}, {3, 6, 7},
		// Bottom face
		{0, 4, 5}, {0, 5, 1},
	}

	return &Mesh{
		Vertices: vertices,
		Faces:    faces,
	}
}

// Color constants
var (
	Black   = Color{0, 0, 0, 255}
	White   = Color{255, 255, 255, 255}
	Red     = Color{255, 0, 0, 255}
	Green   = Color{0, 255, 0, 255}
	Blue    = Color{0, 0, 255, 255}
	Yellow  = Color{255, 255, 0, 255}
	Magenta = Color{255, 0, 255, 255}
	Cyan    = Color{0, 255, 255, 255}
)

// NewColor creates a new color.
func NewColor(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// NewColorRGB creates a new color with full alpha.
func NewColorRGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}

// Lerp performs linear interpolation between two colors.
func (c Color) Lerp(other Color, t float64) Color {
	return Color{
		R: uint8(float64(c.R) + t*(float64(other.R)-float64(c.R))),
		G: uint8(float64(c.G) + t*(float64(other.G)-float64(c.G))),
		B: uint8(float64(c.B) + t*(float64(other.B)-float64(c.B))),
		A: uint8(float64(c.A) + t*(float64(other.A)-float64(c.A))),
	}
}
