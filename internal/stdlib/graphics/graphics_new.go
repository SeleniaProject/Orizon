// Package graphics provides comprehensive 2D and 3D graphics capabilities.
// This package includes advanced drawing primitives, image manipulation,
// 3D rendering, shaders, and animation frameworks.
package graphics

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
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

// Canvas Implementation

// NewCanvas creates a new canvas.
func NewCanvas(width, height int) *Canvas {
	canvas := &Canvas{
		Width:  width,
		Height: height,
		Image:  image.NewRGBA(image.Rect(0, 0, width, height)),
	}
	canvas.Context = NewContext2D(canvas)
	return canvas
}

// NewContext2D creates a new 2D context.
func NewContext2D(canvas *Canvas) *Context2D {
	return &Context2D{
		canvas:      canvas,
		fillColor:   Color{255, 255, 255, 255},
		strokeColor: Color{0, 0, 0, 255},
		lineWidth:   1.0,
		font:        Font{Family: "Arial", Size: 12.0},
	}
}

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

// DrawLine draws a line using Bresenham's algorithm.
func (ctx *Context2D) DrawLine(x0, y0, x1, y1 float64) {
	dx := math.Abs(x1 - x0)
	dy := math.Abs(y1 - y0)

	sx := 1.0
	if x0 > x1 {
		sx = -1.0
	}
	sy := 1.0
	if y0 > y1 {
		sy = -1.0
	}

	err := dx - dy
	x, y := x0, y0

	for {
		ctx.setPixel(int(x), int(y), ctx.strokeColor)

		if x == x1 && y == y1 {
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

// DrawCircle draws a circle using Bresenham's circle algorithm.
func (ctx *Context2D) DrawCircle(centerX, centerY, radius float64) {
	x := 0.0
	y := radius
	d := 3 - 2*radius

	ctx.drawCirclePoints(centerX, centerY, x, y)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		ctx.drawCirclePoints(centerX, centerY, x, y)
	}
}

// drawCirclePoints draws the 8 symmetric points of a circle.
func (ctx *Context2D) drawCirclePoints(centerX, centerY, x, y float64) {
	ctx.setPixel(int(centerX+x), int(centerY+y), ctx.strokeColor)
	ctx.setPixel(int(centerX-x), int(centerY+y), ctx.strokeColor)
	ctx.setPixel(int(centerX+x), int(centerY-y), ctx.strokeColor)
	ctx.setPixel(int(centerX-x), int(centerY-y), ctx.strokeColor)
	ctx.setPixel(int(centerX+y), int(centerY+x), ctx.strokeColor)
	ctx.setPixel(int(centerX-y), int(centerY+x), ctx.strokeColor)
	ctx.setPixel(int(centerX+y), int(centerY-x), ctx.strokeColor)
	ctx.setPixel(int(centerX-y), int(centerY-x), ctx.strokeColor)
}

// FillCircle fills a circle.
func (ctx *Context2D) FillCircle(centerX, centerY, radius float64) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				ctx.setPixel(int(centerX+x), int(centerY+y), ctx.fillColor)
			}
		}
	}
}

// DrawRectangle draws a rectangle outline.
func (ctx *Context2D) DrawRectangle(x, y, width, height float64) {
	ctx.DrawLine(x, y, x+width, y)
	ctx.DrawLine(x+width, y, x+width, y+height)
	ctx.DrawLine(x+width, y+height, x, y+height)
	ctx.DrawLine(x, y+height, x, y)
}

// FillRectangle fills a rectangle.
func (ctx *Context2D) FillRectangle(x, y, width, height float64) {
	for i := 0; i < int(width); i++ {
		for j := 0; j < int(height); j++ {
			ctx.setPixel(int(x)+i, int(y)+j, ctx.fillColor)
		}
	}
}

// DrawPolygon draws a polygon outline.
func (ctx *Context2D) DrawPolygon(points []Point) {
	if len(points) < 2 {
		return
	}

	for i := 0; i < len(points); i++ {
		next := (i + 1) % len(points)
		ctx.DrawLine(points[i].X, points[i].Y, points[next].X, points[next].Y)
	}
}

// FillPolygon fills a polygon using scanline algorithm.
func (ctx *Context2D) FillPolygon(points []Point) {
	if len(points) < 3 {
		return
	}

	// Find bounding box
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Scanline fill
	for y := int(minY); y <= int(maxY); y++ {
		intersections := []int{}

		// Find intersections with polygon edges
		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			p1, p2 := points[i], points[j]

			if (p1.Y <= float64(y) && p2.Y > float64(y)) || (p2.Y <= float64(y) && p1.Y > float64(y)) {
				// Calculate intersection point
				x := p1.X + (float64(y)-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				intersections = append(intersections, int(x))
			}
		}

		// Sort intersections
		sort.Ints(intersections)

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

// DrawBezierCurve draws a cubic Bezier curve.
func (ctx *Context2D) DrawBezierCurve(p0, p1, p2, p3 Point) {
	steps := 100
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		point := ctx.bezierPoint(p0, p1, p2, p3, t)
		if i > 0 {
			prevT := float64(i-1) / float64(steps)
			prevPoint := ctx.bezierPoint(p0, p1, p2, p3, prevT)
			ctx.DrawLine(prevPoint.X, prevPoint.Y, point.X, point.Y)
		}
	}
}

// bezierPoint calculates a point on a cubic Bezier curve.
func (ctx *Context2D) bezierPoint(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	x := uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X
	y := uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y

	return Point{X: x, Y: y}
}

// DrawText draws simple text (basic implementation).
func (ctx *Context2D) DrawText(text string, x, y float64) {
	// Basic pixel font implementation
	for i, char := range text {
		ctx.drawChar(char, x+float64(i)*8, y)
	}
}

// drawChar draws a single character (very basic).
func (ctx *Context2D) drawChar(char rune, x, y float64) {
	// This is a very simplified character rendering
	// In practice, you would use a proper font rendering system
	switch char {
	case 'A':
		ctx.DrawLine(x, y+7, x+3, y)
		ctx.DrawLine(x+3, y, x+6, y+7)
		ctx.DrawLine(x+1, y+4, x+5, y+4)
	case 'B':
		ctx.DrawLine(x, y, x, y+7)
		ctx.DrawLine(x, y, x+4, y)
		ctx.DrawLine(x, y+3, x+4, y+3)
		ctx.DrawLine(x, y+7, x+4, y+7)
		ctx.DrawLine(x+4, y, x+4, y+3)
		ctx.DrawLine(x+4, y+3, x+4, y+7)
	// Add more characters as needed
	default:
		// Draw a simple rectangle for unknown characters
		ctx.DrawRectangle(x, y, 6, 7)
	}
}

// setPixel sets a pixel color.
func (ctx *Context2D) setPixel(x, y int, c Color) {
	if x >= 0 && x < ctx.canvas.Width && y >= 0 && y < ctx.canvas.Height {
		ctx.canvas.Image.Set(x, y, color.RGBA{c.R, c.G, c.B, c.A})
	}
}

// SaveImage saves the canvas to a file.
func (canvas *Canvas) SaveImage(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Determine format from filename
	if strings.HasSuffix(filename, ".png") {
		return png.Encode(file, canvas.Image)
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		return jpeg.Encode(file, canvas.Image, nil)
	}

	return errors.New("unsupported image format")
}

// LoadImage loads an image from file.
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
	canvas := NewCanvas(bounds.Dx(), bounds.Dy())
	draw.Draw(canvas.Image, bounds, img, bounds.Min, draw.Src)

	return canvas, nil
}

// 3D Graphics Implementation

// NewVector3 creates a new 3D vector.
func NewVector3(x, y, z float64) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

// Add adds two vectors.
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Sub subtracts two vectors.
func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Mul multiplies vector by scalar.
func (v Vector3) Mul(scalar float64) Vector3 {
	return Vector3{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar}
}

// Dot calculates dot product.
func (v Vector3) Dot(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross calculates cross product.
func (v Vector3) Cross(other Vector3) Vector3 {
	return Vector3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Length calculates vector length.
func (v Vector3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normalize normalizes the vector.
func (v Vector3) Normalize() Vector3 {
	length := v.Length()
	if length == 0 {
		return Vector3{0, 0, 0}
	}
	return Vector3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

// Matrix4x4 Implementation

// NewMatrix4x4 creates an identity matrix.
func NewMatrix4x4() Matrix4x4 {
	m := Matrix4x4{}
	m.M[0] = 1
	m.M[5] = 1
	m.M[10] = 1
	m.M[15] = 1
	return m
}

// Multiply multiplies two matrices.
func (m Matrix4x4) Multiply(other Matrix4x4) Matrix4x4 {
	result := Matrix4x4{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result.M[i*4+j] += m.M[i*4+k] * other.M[k*4+j]
			}
		}
	}
	return result
}

// TransformVector transforms a 3D vector.
func (m Matrix4x4) TransformVector(v Vector3) Vector3 {
	x := m.M[0]*v.X + m.M[1]*v.Y + m.M[2]*v.Z + m.M[3]
	y := m.M[4]*v.X + m.M[5]*v.Y + m.M[6]*v.Z + m.M[7]
	z := m.M[8]*v.X + m.M[9]*v.Y + m.M[10]*v.Z + m.M[11]
	return Vector3{X: x, Y: y, Z: z}
}

// Translation creates a translation matrix.
func Translation(x, y, z float64) Matrix4x4 {
	m := NewMatrix4x4()
	m.M[3] = x
	m.M[7] = y
	m.M[11] = z
	return m
}

// Rotation creates a rotation matrix around Y axis.
func RotationY(angle float64) Matrix4x4 {
	m := NewMatrix4x4()
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	m.M[0] = cos
	m.M[2] = sin
	m.M[8] = -sin
	m.M[10] = cos
	return m
}

// Perspective creates a perspective projection matrix.
func Perspective(fov, aspect, near, far float64) Matrix4x4 {
	m := Matrix4x4{}
	f := 1.0 / math.Tan(fov/2.0)
	m.M[0] = f / aspect
	m.M[5] = f
	m.M[10] = (far + near) / (near - far)
	m.M[11] = (2 * far * near) / (near - far)
	m.M[14] = -1
	return m
}

// LookAt creates a view matrix.
func LookAt(eye, target, up Vector3) Matrix4x4 {
	zAxis := eye.Sub(target).Normalize()
	xAxis := up.Cross(zAxis).Normalize()
	yAxis := zAxis.Cross(xAxis)

	m := Matrix4x4{}
	m.M[0] = xAxis.X
	m.M[1] = yAxis.X
	m.M[2] = zAxis.X
	m.M[3] = 0
	m.M[4] = xAxis.Y
	m.M[5] = yAxis.Y
	m.M[6] = zAxis.Y
	m.M[7] = 0
	m.M[8] = xAxis.Z
	m.M[9] = yAxis.Z
	m.M[10] = zAxis.Z
	m.M[11] = 0
	m.M[12] = -xAxis.Dot(eye)
	m.M[13] = -yAxis.Dot(eye)
	m.M[14] = -zAxis.Dot(eye)
	m.M[15] = 1

	return m
}

// 3D Renderer Implementation

// NewRenderer3D creates a new 3D renderer.
func NewRenderer3D(width, height int) *Renderer3D {
	return &Renderer3D{
		Width:         width,
		Height:        height,
		ColorBuffer:   make([]Color, width*height),
		DepthBuffer:   make([]float64, width*height),
		ViewMatrix:    NewMatrix4x4(),
		ProjectMatrix: Perspective(math.Pi/4, float64(width)/float64(height), 0.1, 100.0),
		Shading:       PhongShading,
		Culling:       true,
	}
}

// ClearBuffers clears color and depth buffers.
func (r *Renderer3D) ClearBuffers(clearColor Color) {
	for i := range r.ColorBuffer {
		r.ColorBuffer[i] = clearColor
		r.DepthBuffer[i] = math.Inf(1)
	}
}

// SetCamera sets the camera for rendering.
func (r *Renderer3D) SetCamera(camera Camera) {
	r.ViewMatrix = LookAt(camera.Position, camera.Target, camera.Up)
	r.ProjectMatrix = Perspective(camera.FOV, camera.Aspect, camera.Near, camera.Far)
}

// RenderMesh renders a 3D mesh.
func (r *Renderer3D) RenderMesh(mesh *Mesh) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Transform vertices
	transformedVertices := make([]Vertex, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		// World to view space
		viewPos := r.ViewMatrix.TransformVector(vertex.Position)
		// View to clip space
		clipPos := r.ProjectMatrix.TransformVector(viewPos)

		transformedVertices[i] = Vertex{
			Position: clipPos,
			Normal:   vertex.Normal,
			UV:       vertex.UV,
			Color:    vertex.Color,
		}
	}

	// Render triangles
	for i := 0; i < len(mesh.Indices); i += 3 {
		v1 := transformedVertices[mesh.Indices[i]]
		v2 := transformedVertices[mesh.Indices[i+1]]
		v3 := transformedVertices[mesh.Indices[i+2]]

		triangle := Triangle{V0: v1, V1: v2, V2: v3}
		r.renderTriangle(triangle)
	}
}

// renderTriangle renders a single triangle.
func (r *Renderer3D) renderTriangle(triangle Triangle) {
	// Convert to screen coordinates
	v1 := r.toScreenSpace(triangle.V0)
	v2 := r.toScreenSpace(triangle.V1)
	v3 := r.toScreenSpace(triangle.V2)

	// Backface culling
	if r.Culling && r.isBackface(v1, v2, v3) {
		return
	}

	// Rasterize triangle
	r.rasterizeTriangle(v1, v2, v3)
}

// toScreenSpace converts clip space to screen space.
func (r *Renderer3D) toScreenSpace(vertex Vertex) Vertex {
	// Perspective divide
	x := vertex.Position.X / vertex.Position.Z
	y := vertex.Position.Y / vertex.Position.Z
	z := vertex.Position.Z

	// Screen space conversion
	screenX := (x + 1) * float64(r.Width) / 2
	screenY := (1 - y) * float64(r.Height) / 2

	return Vertex{
		Position: Vector3{X: screenX, Y: screenY, Z: z},
		Normal:   vertex.Normal,
		UV:       vertex.UV,
		Color:    vertex.Color,
	}
}

// isBackface checks if triangle is facing away.
func (r *Renderer3D) isBackface(v1, v2, v3 Vertex) bool {
	// Calculate cross product of two edges
	edge1 := v2.Position.Sub(v1.Position)
	edge2 := v3.Position.Sub(v1.Position)
	normal := edge1.Cross(edge2)

	// Check if normal points away from camera
	return normal.Z < 0
}

// rasterizeTriangle rasterizes a triangle using barycentric coordinates.
func (r *Renderer3D) rasterizeTriangle(v1, v2, v3 Vertex) {
	// Bounding box
	minX := int(math.Min(math.Min(v1.Position.X, v2.Position.X), v3.Position.X))
	maxX := int(math.Max(math.Max(v1.Position.X, v2.Position.X), v3.Position.X))
	minY := int(math.Min(math.Min(v1.Position.Y, v2.Position.Y), v3.Position.Y))
	maxY := int(math.Max(math.Max(v1.Position.Y, v2.Position.Y), v3.Position.Y))

	// Clamp to screen bounds
	minX = int(math.Max(0, float64(minX)))
	maxX = int(math.Min(float64(r.Width-1), float64(maxX)))
	minY = int(math.Max(0, float64(minY)))
	maxY = int(math.Min(float64(r.Height-1), float64(maxY)))

	// Rasterize
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := Vector3{X: float64(x), Y: float64(y), Z: 0}
			bary := r.barycentric(v1.Position, v2.Position, v3.Position, p)

			if bary.X >= 0 && bary.Y >= 0 && bary.Z >= 0 {
				// Interpolate depth
				depth := bary.X*v1.Position.Z + bary.Y*v2.Position.Z + bary.Z*v3.Position.Z

				// Depth test
				idx := y*r.Width + x
				if depth < r.DepthBuffer[idx] {
					r.DepthBuffer[idx] = depth

					// Interpolate color
					color := r.interpolateColor(v1.Color, v2.Color, v3.Color, bary)
					r.ColorBuffer[idx] = color
				}
			}
		}
	}
}

// barycentric calculates barycentric coordinates.
func (r *Renderer3D) barycentric(a, b, c, p Vector3) Vector3 {
	v0 := c.Sub(a)
	v1 := b.Sub(a)
	v2 := p.Sub(a)

	dot00 := v0.Dot(v0)
	dot01 := v0.Dot(v1)
	dot02 := v0.Dot(v2)
	dot11 := v1.Dot(v1)
	dot12 := v1.Dot(v2)

	invDenom := 1 / (dot00*dot11 - dot01*dot01)
	u := (dot11*dot02 - dot01*dot12) * invDenom
	v := (dot00*dot12 - dot01*dot02) * invDenom
	w := 1 - u - v

	return Vector3{X: w, Y: v, Z: u}
}

// interpolateColor interpolates colors using barycentric coordinates.
func (r *Renderer3D) interpolateColor(c1, c2, c3 Color, bary Vector3) Color {
	r1 := float64(c1.R)*bary.X + float64(c2.R)*bary.Y + float64(c3.R)*bary.Z
	g1 := float64(c1.G)*bary.X + float64(c2.G)*bary.Y + float64(c3.G)*bary.Z
	b1 := float64(c1.B)*bary.X + float64(c2.B)*bary.Y + float64(c3.B)*bary.Z
	a1 := float64(c1.A)*bary.X + float64(c2.A)*bary.Y + float64(c3.A)*bary.Z

	return Color{
		R: uint8(math.Max(0, math.Min(255, r1))),
		G: uint8(math.Max(0, math.Min(255, g1))),
		B: uint8(math.Max(0, math.Min(255, b1))),
		A: uint8(math.Max(0, math.Min(255, a1))),
	}
}

// CopyToCanvas copies the render buffer to a canvas.
func (r *Renderer3D) CopyToCanvas(canvas *Canvas) {
	for y := 0; y < r.Height && y < canvas.Height; y++ {
		for x := 0; x < r.Width && x < canvas.Width; x++ {
			color := r.ColorBuffer[y*r.Width+x]
			canvas.Image.Set(x, y, color.RGBA{color.R, color.G, color.B, color.A})
		}
	}
}

// Animation Implementation

// NewAnimator creates a new animator.
func NewAnimator() *Animator {
	return &Animator{
		Animations: make(map[string]*Animation),
	}
}

// AddAnimation adds an animation.
func (a *Animator) AddAnimation(animation *Animation) {
	a.Animations[animation.Name] = animation
}

// PlayAnimation starts playing an animation.
func (a *Animator) PlayAnimation(name string) {
	if anim, exists := a.Animations[name]; exists {
		anim.Playing = true
		anim.Time = 0
		a.CurrentAnim = name
	}
}

// UpdateAnimations updates all animations.
func (a *Animator) UpdateAnimations(deltaTime float64) {
	if a.CurrentAnim == "" {
		return
	}

	if anim, exists := a.Animations[a.CurrentAnim]; exists && anim.Playing {
		anim.Time += deltaTime

		if anim.Time >= anim.Duration {
			if anim.Loop {
				anim.Time = 0
			} else {
				anim.Playing = false
			}
		}
	}
}

// GetCurrentKeyframe gets the current keyframe for interpolation.
func (a *Animator) GetCurrentKeyframe(animName string) (Keyframe, Keyframe, float64) {
	anim := a.Animations[animName]
	if anim == nil || len(anim.Keyframes) < 2 {
		return Keyframe{}, Keyframe{}, 0
	}

	for i := 0; i < len(anim.Keyframes)-1; i++ {
		if anim.Time >= anim.Keyframes[i].Time && anim.Time <= anim.Keyframes[i+1].Time {
			t := (anim.Time - anim.Keyframes[i].Time) / (anim.Keyframes[i+1].Time - anim.Keyframes[i].Time)
			return anim.Keyframes[i], anim.Keyframes[i+1], t
		}
	}

	return anim.Keyframes[len(anim.Keyframes)-1], anim.Keyframes[len(anim.Keyframes)-1], 0
}

// Particle System Implementation

// NewParticleSystem creates a new particle system.
func NewParticleSystem(maxParticles int) *ParticleSystem {
	return &ParticleSystem{
		Particles:    make([]Particle, 0, maxParticles),
		MaxParticles: maxParticles,
		EmissionRate: 10.0,
		Gravity:      Vector3{0, -9.81, 0},
		StartColor:   Color{255, 255, 255, 255},
		EndColor:     Color{255, 255, 255, 0},
		StartSize:    1.0,
		EndSize:      0.1,
		ParticleLife: 3.0,
	}
}

// UpdateParticles updates the particle system.
func (ps *ParticleSystem) UpdateParticles(deltaTime float64) {
	// Update existing particles
	for i := len(ps.Particles) - 1; i >= 0; i-- {
		particle := &ps.Particles[i]

		// Update life
		particle.Life -= deltaTime
		if particle.Life <= 0 {
			// Remove dead particle
			ps.Particles = append(ps.Particles[:i], ps.Particles[i+1:]...)
			continue
		}

		// Update physics
		particle.Velocity = particle.Velocity.Add(ps.Gravity.Mul(deltaTime))
		particle.Position = particle.Position.Add(particle.Velocity.Mul(deltaTime))

		// Update visual properties
		t := 1.0 - particle.Life/particle.MaxLife
		particle.Size = ps.StartSize*(1-t) + ps.EndSize*t

		// Interpolate color
		particle.Color = Color{
			R: uint8(float64(ps.StartColor.R)*(1-t) + float64(ps.EndColor.R)*t),
			G: uint8(float64(ps.StartColor.G)*(1-t) + float64(ps.EndColor.G)*t),
			B: uint8(float64(ps.StartColor.B)*(1-t) + float64(ps.EndColor.B)*t),
			A: uint8(float64(ps.StartColor.A)*(1-t) + float64(ps.EndColor.A)*t),
		}
	}

	// Emit new particles
	if len(ps.Particles) < ps.MaxParticles {
		toEmit := int(ps.EmissionRate * deltaTime)
		for i := 0; i < toEmit && len(ps.Particles) < ps.MaxParticles; i++ {
			ps.emitParticle()
		}
	}
}

// emitParticle creates a new particle.
func (ps *ParticleSystem) emitParticle() {
	particle := Particle{
		Position: ps.EmitterPosition,
		Velocity: Vector3{
			X: (math.Random() - 0.5) * 2,
			Y: math.Random() * 2,
			Z: (math.Random() - 0.5) * 2,
		},
		Color:   ps.StartColor,
		Size:    ps.StartSize,
		Life:    ps.ParticleLife,
		MaxLife: ps.ParticleLife,
	}

	ps.Particles = append(ps.Particles, particle)
}

// RenderParticles renders particles to a canvas.
func (ps *ParticleSystem) RenderParticles(ctx *Context2D, camera Camera) {
	for _, particle := range ps.Particles {
		// Simple 2D projection
		x := particle.Position.X + float64(ctx.canvas.Width/2)
		y := particle.Position.Y + float64(ctx.canvas.Height/2)

		ctx.SetFillColor(particle.Color)
		ctx.FillCircle(x, y, particle.Size)
	}
}

// Texture Implementation

// NewTexture creates a new texture.
func NewTexture(width, height int) *Texture {
	return &Texture{
		Width:     width,
		Height:    height,
		Data:      make([]Color, width*height),
		Filtering: FilterLinear,
		Wrapping:  WrapRepeat,
	}
}

// SampleTexture samples a texture at UV coordinates.
func (t *Texture) SampleTexture(u, v float64) Color {
	switch t.Filtering {
	case FilterNearest:
		return t.sampleNearest(u, v)
	case FilterLinear:
		return t.sampleLinear(u, v)
	default:
		return t.sampleNearest(u, v)
	}
}

// sampleNearest samples using nearest neighbor filtering.
func (t *Texture) sampleNearest(u, v float64) Color {
	u, v = t.wrapUV(u, v)
	x := int(u * float64(t.Width-1))
	y := int(v * float64(t.Height-1))
	return t.Data[y*t.Width+x]
}

// sampleLinear samples using bilinear filtering.
func (t *Texture) sampleLinear(u, v float64) Color {
	u, v = t.wrapUV(u, v)

	x := u * float64(t.Width-1)
	y := v * float64(t.Height-1)

	x1 := int(x)
	y1 := int(y)
	x2 := x1 + 1
	y2 := y1 + 1

	if x2 >= t.Width {
		x2 = t.Width - 1
	}
	if y2 >= t.Height {
		y2 = t.Height - 1
	}

	dx := x - float64(x1)
	dy := y - float64(y1)

	c1 := t.Data[y1*t.Width+x1]
	c2 := t.Data[y1*t.Width+x2]
	c3 := t.Data[y2*t.Width+x1]
	c4 := t.Data[y2*t.Width+x2]

	// Bilinear interpolation
	r := (1-dx)*(1-dy)*float64(c1.R) + dx*(1-dy)*float64(c2.R) + (1-dx)*dy*float64(c3.R) + dx*dy*float64(c4.R)
	g := (1-dx)*(1-dy)*float64(c1.G) + dx*(1-dy)*float64(c2.G) + (1-dx)*dy*float64(c3.G) + dx*dy*float64(c4.G)
	b := (1-dx)*(1-dy)*float64(c1.B) + dx*(1-dy)*float64(c2.B) + (1-dx)*dy*float64(c3.B) + dx*dy*float64(c4.B)
	a := (1-dx)*(1-dy)*float64(c1.A) + dx*(1-dy)*float64(c2.A) + (1-dx)*dy*float64(c3.A) + dx*dy*float64(c4.A)

	return Color{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}
}

// wrapUV applies texture wrapping to UV coordinates.
func (t *Texture) wrapUV(u, v float64) (float64, float64) {
	switch t.Wrapping {
	case WrapRepeat:
		u = u - math.Floor(u)
		v = v - math.Floor(v)
	case WrapClamp:
		u = math.Max(0, math.Min(1, u))
		v = math.Max(0, math.Min(1, v))
	case WrapMirror:
		u = math.Abs(u - math.Floor(u))
		v = math.Abs(v - math.Floor(v))
	}
	return u, v
}

// Helper functions

// ColorToRGBA converts Color to color.RGBA.
func (c Color) RGBA() color.RGBA {
	return color.RGBA{c.R, c.G, c.B, c.A}
}

// RGBAToColor converts color.RGBA to Color.
func RGBAToColor(c color.RGBA) Color {
	return Color{c.R, c.G, c.B, c.A}
}

// LerpColor linearly interpolates between two colors.
func LerpColor(c1, c2 Color, t float64) Color {
	return Color{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: uint8(float64(c1.A)*(1-t) + float64(c2.A)*t),
	}
}

// math.Random() replacement since it doesn't exist in standard Go
func random() float64 {
	return float64(rand.Intn(1000)) / 1000.0
}
