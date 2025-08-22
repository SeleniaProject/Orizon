// Package gui provides cross-platform graphical user interface capabilities.
// This package includes window management, widgets, event handling,
// and layout management for desktop applications.
package gui

import (
	"image"
	"image/color"
	"time"
)

// Event represents a GUI event.
type Event interface {
	Type() EventType
	Timestamp() time.Time
}

// EventType represents different types of GUI events.
type EventType int

const (
	MouseEventTypeConst EventType = iota
	KeyboardEventTypeConst
	WindowEventTypeConst
	PaintEventTypeConst
	TimerEventTypeConst
	CustomEventTypeConst
)

// MouseEventSubType represents mouse event subtypes.
type MouseEventSubType int

const (
	MousePress MouseEventSubType = iota
	MouseRelease
	MouseMove
	MouseWheel
	MouseEnter
	MouseLeave
)

// KeyEventSubType represents keyboard event subtypes.
type KeyEventSubType int

const (
	KeyPress KeyEventSubType = iota
	KeyRelease
	KeyRepeat
)

// WindowEventSubType represents window event subtypes.
type WindowEventSubType int

const (
	WindowResize WindowEventSubType = iota
	WindowClose
	WindowMove
	WindowFocus
	WindowBlur
	WindowMinimize
	WindowMaximize
	WindowRestore
)

// Point represents a 2D coordinate.
type Point struct {
	X, Y int
}

// Size represents 2D dimensions.
type Size struct {
	Width, Height int
}

// Rect represents a rectangle.
type Rect struct {
	X, Y, Width, Height int
}

// Color represents an RGBA color.
type Color struct {
	R, G, B, A uint8
}

// MouseEvent represents a mouse event.
type MouseEvent struct {
	EventType MouseEventSubType
	Button    MouseButton
	Position  Point
	Delta     Point // For wheel events
	Modifiers KeyModifiers
	timestamp time.Time
}

// KeyEvent represents a keyboard event.
type KeyEvent struct {
	EventType KeyEventSubType
	Key       Key
	Rune      rune
	Modifiers KeyModifiers
	timestamp time.Time
}

// WindowEvent represents a window event.
type WindowEvent struct {
	EventType WindowEventSubType
	Size      Size
	Position  Point
	timestamp time.Time
}

// MouseButton represents mouse buttons.
type MouseButton int

const (
	LeftButton MouseButton = iota
	RightButton
	MiddleButton
	Button4
	Button5
)

// Key represents keyboard keys.
type Key int

const (
	KeyUnknown Key = iota
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeySpace
	KeyEnter
	KeyTab
	KeyBackspace
	KeyDelete
	KeyInsert
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyEscape
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyShift
	KeyControl
	KeyAlt
	KeyMeta
)

// KeyModifiers represents keyboard modifiers.
type KeyModifiers int

const (
	ModShift KeyModifiers = 1 << iota
	ModControl
	ModAlt
	ModMeta
)

// Event methods
func (e *MouseEvent) Type() EventType      { return MouseEventTypeConst }
func (e *MouseEvent) Timestamp() time.Time { return e.timestamp }

func (e *KeyEvent) Type() EventType      { return KeyboardEventTypeConst }
func (e *KeyEvent) Timestamp() time.Time { return e.timestamp }

func (e *WindowEvent) Type() EventType      { return WindowEventTypeConst }
func (e *WindowEvent) Timestamp() time.Time { return e.timestamp }

// Widget represents a GUI widget.
type Widget interface {
	// Layout and positioning
	Position() Point
	SetPosition(Point)
	Size() Size
	SetSize(Size)
	Bounds() Rect

	// Visibility and state
	Visible() bool
	SetVisible(bool)
	Enabled() bool
	SetEnabled(bool)

	// Event handling
	HandleEvent(Event) bool

	// Rendering
	Paint(*Canvas)

	// Hierarchy
	Parent() Widget
	SetParent(Widget)
	Children() []Widget
	AddChild(Widget)
	RemoveChild(Widget)

	// Focus
	CanFocus() bool
	HasFocus() bool
	SetFocus(bool)
}

// BaseWidget provides common widget functionality.
type BaseWidget struct {
	position Point
	size     Size
	visible  bool
	enabled  bool
	parent   Widget
	children []Widget
	focused  bool
}

// Window represents a top-level window.
type Window struct {
	BaseWidget
	title       string
	resizable   bool
	minimizable bool
	maximizable bool
	closable    bool
	menuBar     *MenuBar
	statusBar   *StatusBar
	canvas      *Canvas
	eventQueue  chan Event
}

// Canvas represents a drawing surface.
type Canvas struct {
	Image    *image.RGBA
	Graphics *Graphics
}

// Graphics provides drawing operations.
type Graphics struct {
	canvas    *Canvas
	fillColor Color
	lineColor Color
	lineWidth int
	font      *Font
}

// Font represents a font.
type Font struct {
	Family string
	Size   int
	Style  FontStyle
}

// FontStyle represents font styling.
type FontStyle int

const (
	FontNormal FontStyle = iota
	FontBold
	FontItalic
	FontBoldItalic
)

// Layout represents a layout manager.
type Layout interface {
	Layout(container Widget)
	MinimumSize(container Widget) Size
	PreferredSize(container Widget) Size
}

// BorderLayout implements border layout.
type BorderLayout struct {
	hgap, vgap int
	north      Widget
	south      Widget
	east       Widget
	west       Widget
	center     Widget
}

// GridLayout implements grid layout.
type GridLayout struct {
	rows, cols int
	hgap, vgap int
}

// FlowLayout implements flow layout.
type FlowLayout struct {
	direction FlowDirection
	hgap      int
	vgap      int
}

// FlowDirection represents flow direction.
type FlowDirection int

const (
	FlowLeftToRight FlowDirection = iota
	FlowRightToLeft
	FlowTopToBottom
	FlowBottomToTop
)

// Common widgets

// Label represents a text label.
type Label struct {
	BaseWidget
	text      string
	textColor Color
	font      *Font
	alignment TextAlignment
}

// Button represents a clickable button.
type Button struct {
	BaseWidget
	text    string
	pressed bool
	onClick func()
	onHover func()
	onLeave func()
}

// TextEdit represents a text input field.
type TextEdit struct {
	BaseWidget
	text        string
	placeholder string
	cursorPos   int
	selection   Range
	multiline   bool
	readonly    bool
	onChange    func(string)
	onEnter     func()
}

// CheckBox represents a checkbox.
type CheckBox struct {
	BaseWidget
	text     string
	checked  bool
	onChange func(bool)
}

// RadioButton represents a radio button.
type RadioButton struct {
	BaseWidget
	text     string
	checked  bool
	group    *RadioGroup
	onChange func(bool)
}

// RadioGroup manages a group of radio buttons.
type RadioGroup struct {
	buttons  []*RadioButton
	selected int
}

// ComboBox represents a dropdown combobox.
type ComboBox struct {
	BaseWidget
	items        []string
	selected     int
	editable     bool
	dropdownOpen bool
	onChange     func(int, string)
}

// ListView represents a list view.
type ListView struct {
	BaseWidget
	items       []string
	selected    []int
	multiSelect bool
	onChange    func([]int)
}

// TreeView represents a tree view.
type TreeView struct {
	BaseWidget
	root     *TreeNode
	selected *TreeNode
	onChange func(*TreeNode)
}

// TreeNode represents a tree node.
type TreeNode struct {
	text     string
	data     interface{}
	parent   *TreeNode
	children []*TreeNode
	expanded bool
}

// MenuBar represents a menu bar.
type MenuBar struct {
	BaseWidget
	menus []*Menu
}

// Menu represents a menu.
type Menu struct {
	text  string
	items []*MenuItem
}

// MenuItem represents a menu item.
type MenuItem struct {
	text      string
	shortcut  string
	enabled   bool
	checkable bool
	checked   bool
	separator bool
	submenu   *Menu
	onClick   func()
}

// StatusBar represents a status bar.
type StatusBar struct {
	BaseWidget
	text     string
	sections []StatusSection
}

// StatusSection represents a status bar section.
type StatusSection struct {
	text  string
	width int
}

// TextAlignment represents text alignment.
type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
)

// Range represents a text range.
type Range struct {
	Start, End int
}

// Application represents the GUI application.
type Application struct {
	windows   []*Window
	mainLoop  bool
	eventChan chan Event
	timers    []*Timer
}

// Timer represents a timer.
type Timer struct {
	interval time.Duration
	callback func()
	running  bool
	ticker   *time.Ticker
	stop     chan bool
}

// Application methods

// NewApplication creates a new GUI application.
func NewApplication() *Application {
	return &Application{
		windows:   make([]*Window, 0),
		mainLoop:  false,
		eventChan: make(chan Event, 100),
		timers:    make([]*Timer, 0),
	}
}

// CreateWindow creates a new window.
func (app *Application) CreateWindow(title string, width, height int) *Window {
	window := &Window{
		BaseWidget: BaseWidget{
			position: Point{100, 100},
			size:     Size{width, height},
			visible:  true,
			enabled:  true,
			children: make([]Widget, 0),
		},
		title:       title,
		resizable:   true,
		minimizable: true,
		maximizable: true,
		closable:    true,
		eventQueue:  make(chan Event, 50),
	}

	// Create canvas
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	window.canvas = &Canvas{
		Image: img,
	}
	window.canvas.Graphics = &Graphics{
		canvas:    window.canvas,
		fillColor: Color{255, 255, 255, 255}, // White
		lineColor: Color{0, 0, 0, 255},       // Black
		lineWidth: 1,
	}

	app.windows = append(app.windows, window)
	return window
}

// Run starts the application main loop.
func (app *Application) Run() error {
	app.mainLoop = true

	for app.mainLoop {
		// Process events
		select {
		case event := <-app.eventChan:
			app.processEvent(event)
		case <-time.After(16 * time.Millisecond): // ~60 FPS
			// Update and render
			app.update()
			app.render()
		}
	}

	return nil
}

// Quit stops the application.
func (app *Application) Quit() {
	app.mainLoop = false
}

// CreateTimer creates a new timer.
func (app *Application) CreateTimer(interval time.Duration, callback func()) *Timer {
	timer := &Timer{
		interval: interval,
		callback: callback,
		running:  false,
		stop:     make(chan bool),
	}

	app.timers = append(app.timers, timer)
	return timer
}

func (app *Application) processEvent(event Event) {
	// Distribute event to appropriate window
	for _, window := range app.windows {
		if window.Visible() {
			if window.HandleEvent(event) {
				break
			}
		}
	}
}

func (app *Application) update() {
	// Update application state
	for _, window := range app.windows {
		if window.Visible() {
			window.update()
		}
	}
}

func (app *Application) render() {
	// Render all windows
	for _, window := range app.windows {
		if window.Visible() {
			window.render()
		}
	}
}

// BaseWidget methods

func (w *BaseWidget) Position() Point     { return w.position }
func (w *BaseWidget) SetPosition(p Point) { w.position = p }
func (w *BaseWidget) Size() Size          { return w.size }
func (w *BaseWidget) SetSize(s Size)      { w.size = s }
func (w *BaseWidget) Bounds() Rect {
	return Rect{w.position.X, w.position.Y, w.size.Width, w.size.Height}
}
func (w *BaseWidget) Visible() bool      { return w.visible }
func (w *BaseWidget) SetVisible(v bool)  { w.visible = v }
func (w *BaseWidget) Enabled() bool      { return w.enabled }
func (w *BaseWidget) SetEnabled(e bool)  { w.enabled = e }
func (w *BaseWidget) Parent() Widget     { return w.parent }
func (w *BaseWidget) SetParent(p Widget) { w.parent = p }
func (w *BaseWidget) Children() []Widget { return w.children }
func (w *BaseWidget) CanFocus() bool     { return false }
func (w *BaseWidget) HasFocus() bool     { return w.focused }
func (w *BaseWidget) SetFocus(f bool)    { w.focused = f }

func (w *BaseWidget) AddChild(child Widget) {
	w.children = append(w.children, child)
	child.SetParent(w)
}

func (w *BaseWidget) RemoveChild(child Widget) {
	for i, c := range w.children {
		if c == child {
			w.children = append(w.children[:i], w.children[i+1:]...)
			child.SetParent(nil)
			break
		}
	}
}

func (w *BaseWidget) HandleEvent(event Event) bool {
	// Pass to children first
	for _, child := range w.children {
		if child.HandleEvent(event) {
			return true
		}
	}
	return false
}

func (w *BaseWidget) Paint(canvas *Canvas) {
	// Paint children
	for _, child := range w.children {
		if child.Visible() {
			child.Paint(canvas)
		}
	}
}

// Window methods

func (w *Window) SetTitle(title string) {
	w.title = title
}

func (w *Window) Title() string {
	return w.title
}

func (w *Window) SetMenuBar(menuBar *MenuBar) {
	w.menuBar = menuBar
}

func (w *Window) MenuBar() *MenuBar {
	return w.menuBar
}

func (w *Window) SetStatusBar(statusBar *StatusBar) {
	w.statusBar = statusBar
}

func (w *Window) StatusBar() *StatusBar {
	return w.statusBar
}

func (w *Window) Canvas() *Canvas {
	return w.canvas
}

func (w *Window) Show() {
	w.SetVisible(true)
}

func (w *Window) Hide() {
	w.SetVisible(false)
}

func (w *Window) Close() {
	w.Hide()
	// Remove from application
}

func (w *Window) update() {
	// Update window state
}

func (w *Window) render() {
	// Clear canvas
	w.canvas.Graphics.Clear(Color{255, 255, 255, 255})

	// Paint all children
	w.Paint(w.canvas)
}

// Graphics methods

func (g *Graphics) SetFillColor(color Color) {
	g.fillColor = color
}

func (g *Graphics) SetLineColor(color Color) {
	g.lineColor = color
}

func (g *Graphics) SetLineWidth(width int) {
	g.lineWidth = width
}

func (g *Graphics) SetFont(font *Font) {
	g.font = font
}

func (g *Graphics) Clear(color Color) {
	bounds := g.canvas.Image.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			g.canvas.Image.Set(x, y, g.colorToRGBA(color))
		}
	}
}

func (g *Graphics) FillRect(x, y, width, height int) {
	for py := y; py < y+height; py++ {
		for px := x; px < x+width; px++ {
			if px >= 0 && py >= 0 && px < g.canvas.Image.Bounds().Dx() && py < g.canvas.Image.Bounds().Dy() {
				g.canvas.Image.Set(px, py, g.colorToRGBA(g.fillColor))
			}
		}
	}
}

func (g *Graphics) DrawRect(x, y, width, height int) {
	// Top edge
	for px := x; px < x+width; px++ {
		g.setPixel(px, y, g.lineColor)
	}
	// Bottom edge
	for px := x; px < x+width; px++ {
		g.setPixel(px, y+height-1, g.lineColor)
	}
	// Left edge
	for py := y; py < y+height; py++ {
		g.setPixel(x, py, g.lineColor)
	}
	// Right edge
	for py := y; py < y+height; py++ {
		g.setPixel(x+width-1, py, g.lineColor)
	}
}

func (g *Graphics) DrawLine(x1, y1, x2, y2 int) {
	// Bresenham's line algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

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
	x, y := x1, y1

	for {
		g.setPixel(x, y, g.lineColor)

		if x == x2 && y == y2 {
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

func (g *Graphics) DrawText(text string, x, y int) {
	// Simple text rendering
	for i, char := range text {
		g.drawChar(char, x+i*8, y)
	}
}

func (g *Graphics) drawChar(char rune, x, y int) {
	// Very basic character rendering
	switch char {
	case 'A':
		g.DrawLine(x, y+8, x+4, y)
		g.DrawLine(x+4, y, x+8, y+8)
		g.DrawLine(x+2, y+4, x+6, y+4)
	case 'B':
		g.DrawLine(x, y, x, y+8)
		g.DrawLine(x, y, x+6, y)
		g.DrawLine(x, y+4, x+6, y+4)
		g.DrawLine(x, y+8, x+6, y+8)
		g.DrawLine(x+6, y, x+6, y+4)
		g.DrawLine(x+6, y+4, x+6, y+8)
	default:
		g.DrawRect(x, y, 8, 8)
	}
}

func (g *Graphics) setPixel(x, y int, color Color) {
	if x >= 0 && y >= 0 && x < g.canvas.Image.Bounds().Dx() && y < g.canvas.Image.Bounds().Dy() {
		g.canvas.Image.Set(x, y, g.colorToRGBA(color))
	}
}

func (g *Graphics) colorToRGBA(c Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Label methods

func NewLabel(text string) *Label {
	return &Label{
		BaseWidget: BaseWidget{
			visible: true,
			enabled: true,
		},
		text:      text,
		textColor: Color{0, 0, 0, 255},
		alignment: AlignLeft,
	}
}

func (l *Label) SetText(text string) {
	l.text = text
}

func (l *Label) Text() string {
	return l.text
}

func (l *Label) SetTextColor(color Color) {
	l.textColor = color
}

func (l *Label) SetAlignment(align TextAlignment) {
	l.alignment = align
}

func (l *Label) Paint(canvas *Canvas) {
	bounds := l.Bounds()

	// Draw text
	var x int
	switch l.alignment {
	case AlignLeft:
		x = bounds.X
	case AlignCenter:
		x = bounds.X + (bounds.Width-len(l.text)*8)/2
	case AlignRight:
		x = bounds.X + bounds.Width - len(l.text)*8
	}

	oldColor := canvas.Graphics.lineColor
	canvas.Graphics.SetLineColor(l.textColor)
	canvas.Graphics.DrawText(l.text, x, bounds.Y)
	canvas.Graphics.SetLineColor(oldColor)

	l.BaseWidget.Paint(canvas)
}

// Button methods

func NewButton(text string) *Button {
	return &Button{
		BaseWidget: BaseWidget{
			visible: true,
			enabled: true,
		},
		text: text,
	}
}

func (b *Button) SetText(text string) {
	b.text = text
}

func (b *Button) Text() string {
	return b.text
}

func (b *Button) SetOnClick(callback func()) {
	b.onClick = callback
}

func (b *Button) CanFocus() bool {
	return true
}

func (b *Button) HandleEvent(event Event) bool {
	if !b.Enabled() {
		return false
	}

	switch e := event.(type) {
	case *MouseEvent:
		bounds := b.Bounds()
		if e.Position.X >= bounds.X && e.Position.X < bounds.X+bounds.Width &&
			e.Position.Y >= bounds.Y && e.Position.Y < bounds.Y+bounds.Height {

			switch e.EventType {
			case MousePress:
				if e.Button == LeftButton {
					b.pressed = true
					return true
				}
			case MouseRelease:
				if e.Button == LeftButton && b.pressed {
					b.pressed = false
					if b.onClick != nil {
						b.onClick()
					}
					return true
				}
			}
		} else {
			b.pressed = false
		}
	}

	return b.BaseWidget.HandleEvent(event)
}

func (b *Button) Paint(canvas *Canvas) {
	bounds := b.Bounds()

	// Draw button background
	if b.pressed {
		canvas.Graphics.SetFillColor(Color{200, 200, 200, 255})
	} else {
		canvas.Graphics.SetFillColor(Color{240, 240, 240, 255})
	}
	canvas.Graphics.FillRect(bounds.X, bounds.Y, bounds.Width, bounds.Height)

	// Draw border
	canvas.Graphics.SetLineColor(Color{128, 128, 128, 255})
	canvas.Graphics.DrawRect(bounds.X, bounds.Y, bounds.Width, bounds.Height)

	// Draw text
	textX := bounds.X + (bounds.Width-len(b.text)*8)/2
	textY := bounds.Y + (bounds.Height-8)/2
	if b.pressed {
		textX++
		textY++
	}

	canvas.Graphics.SetLineColor(Color{0, 0, 0, 255})
	canvas.Graphics.DrawText(b.text, textX, textY)

	b.BaseWidget.Paint(canvas)
}

// Timer methods

func (t *Timer) Start() {
	if t.running {
		return
	}

	t.running = true
	t.ticker = time.NewTicker(t.interval)

	go func() {
		for {
			select {
			case <-t.ticker.C:
				if t.callback != nil {
					t.callback()
				}
			case <-t.stop:
				return
			}
		}
	}()
}

func (t *Timer) Stop() {
	if !t.running {
		return
	}

	t.running = false
	if t.ticker != nil {
		t.ticker.Stop()
	}
	t.stop <- true
}

func (t *Timer) IsRunning() bool {
	return t.running
}

// Layout implementations

func NewBorderLayout(hgap, vgap int) *BorderLayout {
	return &BorderLayout{
		hgap: hgap,
		vgap: vgap,
	}
}

func (bl *BorderLayout) AddWidget(widget Widget, position string) {
	switch position {
	case "north":
		bl.north = widget
	case "south":
		bl.south = widget
	case "east":
		bl.east = widget
	case "west":
		bl.west = widget
	case "center":
		bl.center = widget
	}
}

func (bl *BorderLayout) Layout(container Widget) {
	bounds := container.Bounds()

	northHeight := 0
	southHeight := 0
	eastWidth := 0
	westWidth := 0

	// Calculate sizes
	if bl.north != nil && bl.north.Visible() {
		northHeight = bl.north.Size().Height
	}
	if bl.south != nil && bl.south.Visible() {
		southHeight = bl.south.Size().Height
	}
	if bl.east != nil && bl.east.Visible() {
		eastWidth = bl.east.Size().Width
	}
	if bl.west != nil && bl.west.Visible() {
		westWidth = bl.west.Size().Width
	}

	// Layout widgets
	if bl.north != nil && bl.north.Visible() {
		bl.north.SetPosition(Point{bounds.X, bounds.Y})
		bl.north.SetSize(Size{bounds.Width, northHeight})
	}

	if bl.south != nil && bl.south.Visible() {
		bl.south.SetPosition(Point{bounds.X, bounds.Y + bounds.Height - southHeight})
		bl.south.SetSize(Size{bounds.Width, southHeight})
	}

	if bl.west != nil && bl.west.Visible() {
		bl.west.SetPosition(Point{bounds.X, bounds.Y + northHeight + bl.vgap})
		bl.west.SetSize(Size{westWidth, bounds.Height - northHeight - southHeight - 2*bl.vgap})
	}

	if bl.east != nil && bl.east.Visible() {
		bl.east.SetPosition(Point{bounds.X + bounds.Width - eastWidth, bounds.Y + northHeight + bl.vgap})
		bl.east.SetSize(Size{eastWidth, bounds.Height - northHeight - southHeight - 2*bl.vgap})
	}

	if bl.center != nil && bl.center.Visible() {
		centerX := bounds.X + westWidth + bl.hgap
		centerY := bounds.Y + northHeight + bl.vgap
		centerWidth := bounds.Width - westWidth - eastWidth - 2*bl.hgap
		centerHeight := bounds.Height - northHeight - southHeight - 2*bl.vgap

		bl.center.SetPosition(Point{centerX, centerY})
		bl.center.SetSize(Size{centerWidth, centerHeight})
	}
}

func (bl *BorderLayout) MinimumSize(container Widget) Size {
	return Size{100, 100}
}

func (bl *BorderLayout) PreferredSize(container Widget) Size {
	return Size{300, 200}
}

// Color constants
var (
	ColorBlack   = Color{0, 0, 0, 255}
	ColorWhite   = Color{255, 255, 255, 255}
	ColorRed     = Color{255, 0, 0, 255}
	ColorGreen   = Color{0, 255, 0, 255}
	ColorBlue    = Color{0, 0, 255, 255}
	ColorYellow  = Color{255, 255, 0, 255}
	ColorMagenta = Color{255, 0, 255, 255}
	ColorCyan    = Color{0, 255, 255, 255}
	ColorGray    = Color{128, 128, 128, 255}
)

// NewColor creates a new color.
func NewColor(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// NewColorRGB creates a new color with full alpha.
func NewColorRGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}
